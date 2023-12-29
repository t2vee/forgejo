// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"testing"

	"code.gitea.io/gitea/modules/setting"
)

func TestNewPersonId(t *testing.T) {
	expected := PersonID{}
	expected.ID = "1"
	expected.Source = "forgejo"
	expected.Schema = "https"
	expected.Path = "api/v1/activitypub/user-id"
	expected.Host = "an.other.host"
	expected.Port = ""
	expected.UnvalidatedInput = "https://an.other.host/api/v1/activitypub/user-id/1"
	sut, _ := NewPersonID("https://an.other.host/api/v1/activitypub/user-id/1", "forgejo")
	if sut != expected {
		t.Errorf("expected: %v\n but was: %v\n", expected, sut)
	}

	expected = PersonID{}
	expected.ID = "1"
	expected.Source = "forgejo"
	expected.Schema = "https"
	expected.Path = "api/v1/activitypub/user-id"
	expected.Host = "an.other.host"
	expected.Port = "443"
	expected.UnvalidatedInput = "https://an.other.host:443/api/v1/activitypub/user-id/1"
	sut, _ = NewPersonID("https://an.other.host:443/api/v1/activitypub/user-id/1", "forgejo")
	if sut != expected {
		t.Errorf("expected: %v\n but was: %v\n", expected, sut)
	}
}

func TestNewRepositoryId(t *testing.T) {
	setting.AppURL = "http://localhost:3000/"
	expected := RepositoryID{}
	expected.ID = "1"
	expected.Source = "forgejo"
	expected.Schema = "http"
	expected.Path = "api/activitypub/repository-id"
	expected.Host = "localhost"
	expected.Port = "3000"
	expected.UnvalidatedInput = "http://localhost:3000/api/activitypub/repository-id/1"
	sut, _ := NewRepositoryID("http://localhost:3000/api/activitypub/repository-id/1", "forgejo")
	if sut != expected {
		t.Errorf("expected: %v\n but was: %v\n", expected, sut)
	}
}

func TestActorIdValidation(t *testing.T) {
	sut := ActorID{}
	sut.Source = "forgejo"
	sut.Schema = "https"
	sut.Path = "api/v1/activitypub/user-id"
	sut.Host = "an.other.host"
	sut.Port = ""
	sut.UnvalidatedInput = "https://an.other.host/api/v1/activitypub/user-id/"
	if sut.Validate()[0] != "Field userId may not be empty" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate())
	}

	sut = ActorID{}
	sut.ID = "1"
	sut.Source = "forgejo"
	sut.Schema = "https"
	sut.Path = "api/v1/activitypub/user-id"
	sut.Host = "an.other.host"
	sut.Port = ""
	sut.UnvalidatedInput = "https://an.other.host/api/v1/activitypub/user-id/1?illegal=action"
	if sut.Validate()[0] != "not all input: \"https://an.other.host/api/v1/activitypub/user-id/1?illegal=action\" was parsed: \"https://an.other.host/api/v1/activitypub/user-id/1\"" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate())
	}
}

func TestPersonIdValidation(t *testing.T) {
	sut := PersonID{}
	sut.ID = "1"
	sut.Source = "forgejo"
	sut.Schema = "https"
	sut.Path = "path"
	sut.Host = "an.other.host"
	sut.Port = ""
	sut.UnvalidatedInput = "https://an.other.host/path/1"
	if _, err := IsValid(sut); err.Error() != "path: \"path\" has to be a person specific api path" {
		t.Errorf("validation error expected but was: %v\n", err)
	}

	sut = PersonID{}
	sut.ID = "1"
	sut.Source = "forgejox"
	sut.Schema = "https"
	sut.Path = "api/v1/activitypub/user-id"
	sut.Host = "an.other.host"
	sut.Port = ""
	sut.UnvalidatedInput = "https://an.other.host/api/v1/activitypub/user-id/1"
	if sut.Validate()[0] != "Value forgejox is not contained in allowed values [[forgejo gitea]]" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate())
	}
}

func TestWebfingerId(t *testing.T) {
	sut, _ := NewPersonID("https://codeberg.org/api/v1/activitypub/user-id/12345", "forgejo")
	if sut.AsWebfinger() != "@12345@codeberg.org" {
		t.Errorf("wrong webfinger: %v", sut.AsWebfinger())
	}

	sut, _ = NewPersonID("https://Codeberg.org/api/v1/activitypub/user-id/12345", "forgejo")
	if sut.AsWebfinger() != "@12345@codeberg.org" {
		t.Errorf("wrong webfinger: %v", sut.AsWebfinger())
	}
}

func TestShouldThrowErrorOnInvalidInput(t *testing.T) {
	_, err := NewPersonId("", "forgejo")
	if err == nil {
		t.Errorf("empty input should be invalid.")
	}

	_, err = NewPersonID("http://localhost:3000/api/v1/something", "forgejo")
	if err == nil {
		t.Errorf("localhost uris are not external")
	}
	_, err = NewPersonID("./api/v1/something", "forgejo")
	if err == nil {
		t.Errorf("relative uris are not alowed")
	}
	_, err = NewPersonID("http://1.2.3.4/api/v1/something", "forgejo")
	if err == nil {
		t.Errorf("uri may not be ip-4 based")
	}
	_, err = NewPersonID("http:///[fe80::1ff:fe23:4567:890a%25eth0]/api/v1/something", "forgejo")
	if err == nil {
		t.Errorf("uri may not be ip-6 based")
	}
	_, err = NewPersonID("https://codeberg.org/api/v1/activitypub/../activitypub/user-id/12345", "forgejo")
	if err == nil {
		t.Errorf("uri may not contain relative path elements")
	}
	_, err = NewPersonID("https://myuser@an.other.host/api/v1/activitypub/user-id/1", "forgejo")
	if err == nil {
		t.Errorf("uri may not contain unparsed elements")
	}

	_, err = NewPersonID("https://an.other.host/api/v1/activitypub/user-id/1", "forgejo")
	if err != nil {
		t.Errorf("this uri should be valid but was: %v", err)
	}
}
