// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package structs

// QuotaGroupAddOrRemoveUserOption represents the options for quota group membership management
type QuotaGroupAddOrRemoveUserOption struct {
	// Name of the user to add to or remove from the quota group
	// required: true
	Username string `json:"username" binding:"Required"`
}

// QuotaUsedAttachmentList represents a list of attachment counting towards a user's quota
type QuotaUsedAttachmentList []*QuotaUsedAttachment

// QuotaUsedAttachment represents an attachment counting towards a user's quota
type QuotaUsedAttachment struct {
	// Filename of the attachment
	Name string `json:"name"`
	// Size of the attachment (in bytes)
	Size int64 `json:"size"`
	// API URL for the attachment
	APIURL string `json:"api_url"`
	// Context for the attachment: URLs to the containing object
	ContainedIn struct {
		// API URL for the object that contains this attachment
		APIURL string `json:"api_url"`
		// HTML URL for the object that contains this attachment
		HTMLURL string `json:"html_url"`
	} `json:"contained_in"`
}

// QuotaUsedPackageList represents a list of packages counting towards a user's quota
type QuotaUsedPackageList []*QuotaUsedPackage

// QuotaUsedPackage represents a package counting towards a user's quota
type QuotaUsedPackage struct {
	// Name of the package
	Name string `json:"name"`
	// Type of the package
	Type string `json:"type"`
	// Version of the package
	Version string `json:"version"`
	// Size of the package version
	Size int64 `json:"size"`
	// HTML URL to the package version
	HTMLURL string `json:"html_url"`
}

// QuotaUsedArtifactList represents a list of artifacts counting towards a user's quota
type QuotaUsedArtifactList []*QuotaUsedArtifact

// QuotaUsedArtifact represents an artifact counting towards a user's quota
type QuotaUsedArtifact struct {
	// Name of the artifact
	Name string `json:"name"`
	// Size of the artifact (compressed)
	Size int64 `json:"size"`
	// HTML URL to the action run containing the artifact
	HTMLURL string `json:"html_url"`
}
