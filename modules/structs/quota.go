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
