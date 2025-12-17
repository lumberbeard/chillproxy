package core

import (
	"regexp"
)

// uuidRegex matches UUID v4 format: 8-4-4-4-12 hex digits (lowercase only)
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// IsValidUUID validates if a string is a valid UUID v4 (lowercase hex only)
func IsValidUUID(uuid string) bool {
	return uuidRegex.MatchString(uuid)
}

