package validator

import "github.com/google/uuid"

// IsStrictUUIDv4 returns true only when s is a syntactically valid UUID and
// its version is exactly 4 (random). Version bits are checked after parsing so
// that a well-formed v1/v3/v5 string is rejected rather than silently accepted.
func IsStrictUUIDv4(s string) bool {
	parsed, err := uuid.Parse(s)
	return err == nil && parsed.Version() == 4
}
