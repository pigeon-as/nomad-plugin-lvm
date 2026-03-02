package lvm

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple", "myvolume", false},
		{"with dots", "my.volume", false},
		{"with dashes", "my-volume", false},
		{"with underscores", "my_volume", false},
		{"leading dot", ".hidden", false},
		{"leading underscore", "_internal", false},
		{"numeric", "123", false},
		{"empty", "", true},
		{"has slash", "my/volume", true},
		{"has space", "my volume", true},
		{"starts with dash", "-invalid", true},
		{"special chars", "vol@ume!", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateName(tc.input)
			if tc.wantErr {
				must.Error(t, err)
			} else {
				must.NoError(t, err)
			}
		})
	}
}
