// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/validation"

	ap "github.com/go-ap/activitypub"
)

// ForgeLike activity data type
// swagger:model
type ForgeLike struct {
	// swagger:ignore
	ap.Activity
}

// TODO: Use explicit values instead of ctx !!
func NewForgeLike(actorIRI string, objectIRI string) (ForgeLike, error) {
	result := ForgeLike{}
	result.Type = ap.LikeType
	// ToDo: Would validating the source by Actor.Type field make sense?
	object := new(ap.Object)
	object.ID = ap.IRI(objectIRI)

	result.Actor = ap.ActorNew(ap.IRI(actorIRI), "ForgejoUser") // Thats us, a User
	result.Object = object                                      // Thats them, a Repository
	log.Info("Object is: %v", object)
	result.StartTime = time.Now()
	if valid, err := validation.IsValid(result); !valid {
		return ForgeLike{}, err
	}
	return result, nil
}

func (like ForgeLike) MarshalJSON() ([]byte, error) {
	return like.Activity.MarshalJSON()
}

func (like *ForgeLike) UnmarshalJSON(data []byte) error {
	return like.Activity.UnmarshalJSON(data)
}

func (like ForgeLike) IsNewer(compareTo time.Time) bool {
	return like.StartTime.After(compareTo)
}

func (like ForgeLike) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(string(like.Type), "type")...)
	result = append(result, validation.ValidateOneOf(string(like.Type), []any{"Like"})...)
	result = append(result, validation.ValidateNotEmpty(like.Actor.GetID().String(), "actor")...)
	result = append(result, validation.ValidateNotEmpty(like.Object.GetID().String(), "object")...)
	result = append(result, validation.ValidateNotEmpty(like.StartTime.String(), "startTime")...)
	if like.StartTime.IsZero() {
		result = append(result, "StartTime was invalid.")
	}

	return result
}
