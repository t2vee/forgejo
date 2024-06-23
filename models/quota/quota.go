// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package quota

import (
	"context"

 	"code.gitea.io/gitea/models/db"
)

func GetGitUseForUser(ctx context.Context, userID int64) (int64, error) {
	var size int64
	_, err := db.GetEngine(ctx).Select("SUM(size) AS size").
		Table("repository").
		Where("owner_id = ?", userID).
		Get(&size)
	return size, err
}

func GetFilesUseForUser(ctx context.Context, userID int64) (int64, error) {
	var totalSize int64
	var size int64

	_, err := db.GetEngine(ctx).Select("SUM(size) AS size").
		Table("attachment").
		Where("uploader_id = ?", userID).
		Get(&size)
	if err != nil {
		return 0, err
	}
	totalSize += size

	size = 0
	_, err = db.GetEngine(ctx).Select("SUM(file_compressed_size) AS size").
		Table("action_artifact").
		Where("owner_id = ?", userID).
		Get(&size)
	if err != nil {
		return 0, err
	}
	totalSize += size

	size = 0
	_, err = db.GetEngine(ctx).Select("SUM(package_blob.size) AS size").
		Table("package_blob").
		Join("INNER", "`package_file`", "`package_file`.blob_id = `package_blob`.id").
		Join("INNER", "`package_version`", "`package_file`.version_id = `package_version`.id").
		Join("INNER", "`package`", "`package_version`.package_id = `package`.id" ).
		Where("`package`.owner_id = ?", userID).
		Get(&size)
	if err != nil {
		return 0, err
	}
	totalSize += size

	return totalSize, nil
}
