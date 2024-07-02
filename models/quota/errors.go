// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota

import "fmt"

type ErrRuleAlreadyExists struct {
	Name string
}

func IsErrRuleAlreadyExists(err error) bool {
	_, ok := err.(ErrRuleAlreadyExists)
	return ok
}

func (err ErrRuleAlreadyExists) Error() string {
	return fmt.Sprintf("rule already exists: [name: %s]", err.Name)
}

type ErrRuleNotFound struct {
	Name string
}

func IsErrRuleNotFound(err error) bool {
	_, ok := err.(ErrRuleNotFound)
	return ok
}

func (err ErrRuleNotFound) Error() string {
	return fmt.Sprintf("rule not found: [name: %s]", err.Name)
}

type ErrGroupAlreadyExists struct {
	Name string
}

func IsErrGroupAlreadyExists(err error) bool {
	_, ok := err.(ErrGroupAlreadyExists)
	return ok
}

func (err ErrGroupAlreadyExists) Error() string {
	return fmt.Sprintf("group already exists: [name: %s]", err.Name)
}

type ErrGroupNotFound struct {
	Name string
}

func IsErrGroupNotFound(err error) bool {
	_, ok := err.(ErrGroupNotFound)
	return ok
}

func (err ErrGroupNotFound) Error() string {
	return fmt.Sprintf("group not found: [group: %s]", err.Name)
}

type ErrUserAlreadyInGroup struct {
	GroupName string
	UserID    int64
}

func IsErrUserAlreadyInGroup(err error) bool {
	_, ok := err.(ErrUserAlreadyInGroup)
	return ok
}

func (err ErrUserAlreadyInGroup) Error() string {
	return fmt.Sprintf("user already in group: [group: %s, userID: %d]", err.GroupName, err.UserID)
}

type ErrUserNotInGroup struct {
	GroupName string
	UserID    int64
}

func IsErrUserNotInGroup(err error) bool {
	_, ok := err.(ErrUserNotInGroup)
	return ok
}

func (err ErrUserNotInGroup) Error() string {
	return fmt.Sprintf("user not in group: [group: %s, userID: %d]", err.GroupName, err.UserID)
}

type ErrRuleAlreadyInGroup struct {
	GroupName string
	RuleName  string
}

func IsErrRuleAlreadyInGroup(err error) bool {
	_, ok := err.(ErrRuleAlreadyInGroup)
	return ok
}

func (err ErrRuleAlreadyInGroup) Error() string {
	return fmt.Sprintf("rule already in group: [group: %s, rule: %s]", err.GroupName, err.RuleName)
}

type ErrRuleNotInGroup struct {
	GroupName string
	RuleName  string
}

func IsErrRuleNotInGroup(err error) bool {
	_, ok := err.(ErrRuleNotInGroup)
	return ok
}

func (err ErrRuleNotInGroup) Error() string {
	return fmt.Sprintf("rule not in group: [group: %s, rule: %s]", err.GroupName, err.RuleName)
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
