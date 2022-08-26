package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccessTokenScope_Normalize(t *testing.T) {
	tests := []struct {
		in  AccessTokenScope
		out AccessTokenScope
		err error
	}{
		{"", "", nil},
		{"user", "user", nil},
		{"user,read:user", "user", nil},
	}

	for _, test := range tests {
		t.Run(string(test.in), func(t *testing.T) {
			scope, err := test.in.Normalize()
			assert.Equal(t, test.out, scope)
			assert.Equal(t, test.err, err)
		})
	}
}
