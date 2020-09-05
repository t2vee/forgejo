// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package password

import (
	"context"

	"code.gitea.io/gitea/modules/setting"

	"go.jolheiser.com/pwn"
)

// IsPwned checks whether a password has been pwned too many times
// according to threshold
func IsPwned(ctx context.Context, password string) (bool, error) {
	if !setting.PasswordCheckPwn {
		return false, nil
	}

	client := pwn.New(pwn.WithContext(ctx))
	count, err := client.CheckPassword(password, true)
	if err != nil {
		return true, err
	}

	return count > 0, nil
}
