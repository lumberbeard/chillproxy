package core

import "testing"

func TestIsValidUUID_Valid(t *testing.T) {
	validUUIDs := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"00000000-0000-0000-0000-000000000000",
		"ffffffff-ffff-ffff-ffff-ffffffffffff",
		"123e4567-e89b-12d3-a456-426614174000",
	}

	for _, uuid := range validUUIDs {
		if !IsValidUUID(uuid) {
			t.Errorf("Expected %s to be valid UUID", uuid)
		}
	}
}

func TestIsValidUUID_Invalid(t *testing.T) {
	invalidUUIDs := []string{
		"",                                      // Empty
		"not-a-uuid",                           // Wrong format
		"550e8400-e29b-41d4-a716",              // Too short
		"550e8400-e29b-41d4-a716-446655440000-extra", // Too long
		"550e8400-e29b-41d4-a716-44665544000",  // Wrong segment length
		"550e8400e29b41d4a716446655440000",     // No dashes
		"550E8400-E29B-41D4-A716-446655440000", // Uppercase (should be lowercase)
		"550e8400-e29b-41d4-a716-44665544000g", // Invalid character (g)
		"550e8400-e29b-41d4-a71g-446655440000", // Invalid character (g)
		"test-user-id-not-a-uuid",              // Random string
		"12345678-1234-1234-1234-123456789abc", // Has valid hex but wrong count (13 in last segment)
	}

	for _, uuid := range invalidUUIDs {
		if IsValidUUID(uuid) {
			t.Errorf("Expected %s to be invalid UUID", uuid)
		}
	}
}

func TestIsValidUUID_CaseSensitive(t *testing.T) {
	// Only lowercase should be valid
	lowercase := "550e8400-e29b-41d4-a716-446655440000"
	uppercase := "550E8400-E29B-41D4-A716-446655440000"
	mixed := "550e8400-E29B-41d4-A716-446655440000"

	if !IsValidUUID(lowercase) {
		t.Errorf("Expected lowercase UUID to be valid")
	}

	if IsValidUUID(uppercase) {
		t.Errorf("Expected uppercase UUID to be invalid")
	}

	if IsValidUUID(mixed) {
		t.Errorf("Expected mixed case UUID to be invalid")
	}
}

func TestIsValidUUID_Format(t *testing.T) {
	// Test specific format requirements: 8-4-4-4-12
	tests := []struct {
		uuid  string
		valid bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},  // Correct format
		{"550e840-e29b-41d4-a716-446655440000", false},  // 7-4-4-4-12
		{"550e8400-e29-41d4-a716-446655440000", false},  // 8-3-4-4-12
		{"550e8400-e29b-41d-a716-446655440000", false},  // 8-4-3-4-12
		{"550e8400-e29b-41d4-a71-446655440000", false},  // 8-4-4-3-12
		{"550e8400-e29b-41d4-a716-44665544000", false},  // 8-4-4-4-11
	}

	for _, tt := range tests {
		result := IsValidUUID(tt.uuid)
		if result != tt.valid {
			t.Errorf("UUID %s: expected %v, got %v", tt.uuid, tt.valid, result)
		}
	}
}

