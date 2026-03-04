package lvm

import (
	"fmt"
	"regexp"
)

var validNameRe = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]*$`)

// ValidateName checks whether a string is a valid LVM logical volume name.
func ValidateName(name string) error {
	if !validNameRe.MatchString(name) {
		return fmt.Errorf("invalid LV name: %q", name)
	}
	return nil
}
