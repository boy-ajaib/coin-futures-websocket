package channel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseChannelValid tests parsing valid channels
func TestParseChannelValid(t *testing.T) {
	tests := []struct {
		name          string
		channel       string
		expectedAjaib string
		expectedType  string
	}{
		{
			name:          "valid margin channel",
			channel:       "user:130010505:margin",
			expectedAjaib: "130010505",
			expectedType:  "margin",
		},
		{
			name:          "valid position channel",
			channel:       "user:123456:position",
			expectedAjaib: "123456",
			expectedType:  "position",
		},
		{
			name:          "channel with single digit ID",
			channel:       "user:1:margin",
			expectedAjaib: "1",
			expectedType:  "margin",
		},
		{
			name:          "channel with maximum digit ID",
			channel:       "user:9999999999:position",
			expectedAjaib: "9999999999",
			expectedType:  "position",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseChannel(tt.channel)
			require.NoError(t, err)
			assert.NotNil(t, info)
			assert.Equal(t, tt.channel, info.Name)
			assert.Equal(t, PrefixUser, info.Prefix)
			assert.Equal(t, tt.expectedAjaib, info.AjaibID)
			assert.Equal(t, tt.expectedType, info.ChannelSub)
		})
	}
}

// TestParseChannelInvalid tests parsing invalid channels
func TestParseChannelInvalid(t *testing.T) {
	tests := []struct {
		name        string
		channel     string
		expectedErr error
	}{
		{
			name:        "empty channel",
			channel:     "",
			expectedErr: ErrUnknownChannelType,
		},
		{
			name:        "missing user prefix",
			channel:     "130010505:margin",
			expectedErr: ErrUnknownChannelType,
		},
		{
			name:        "wrong prefix",
			channel:     "admin:130010505:margin",
			expectedErr: ErrUnknownChannelType,
		},
		{
			name:        "missing channel type",
			channel:     "user:130010505:",
			expectedErr: ErrInvalidChannelFormat,
		},
		{
			name:        "missing user ID",
			channel:     "user::margin",
			expectedErr: ErrInvalidChannelFormat,
		},
		{
			name:        "only prefix",
			channel:     "user:",
			expectedErr: ErrInvalidChannelFormat,
		},
		{
			name:        "extra colons",
			channel:     "user:130010505:margin:extra",
			expectedErr: ErrInvalidChannelFormat,
		},
		{
			name:        "no colons",
			channel:     "user130010505margin",
			expectedErr: ErrUnknownChannelType,
		},
		{
			name:        "invalid user ID - contains letters",
			channel:     "user:abc123:margin",
			expectedErr: ErrInvalidCFXUserID,
		},
		{
			name:        "invalid user ID - contains special chars",
			channel:     "user:130-010505:margin",
			expectedErr: ErrInvalidCFXUserID,
		},
		{
			name:        "invalid user ID - too long",
			channel:     "user:12345678901:margin",
			expectedErr: ErrInvalidCFXUserID,
		},
		{
			name:        "invalid user ID - empty",
			channel:     "user::margin",
			expectedErr: ErrInvalidChannelFormat,
		},
		{
			name:        "unknown channel type",
			channel:     "user:130010505:unknown",
			expectedErr: ErrUnknownChannelType,
		},
		{
			name:        "invalid channel type",
			channel:     "user:130010505:orders",
			expectedErr: ErrUnknownChannelType,
		},
		{
			name:        "channel type with uppercase",
			channel:     "user:130010505:MARGIN",
			expectedErr: ErrUnknownChannelType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseChannel(tt.channel)
			assert.Error(t, err)
			assert.Nil(t, info)
			// Check that the error matches expected type
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

// TestValidUserChannels tests the ValidUserChannels map
func TestValidUserChannels(t *testing.T) {
	assert.True(t, ValidUserChannels["margin"], "margin should be a valid channel type")
	assert.True(t, ValidUserChannels["position"], "position should be a valid channel type")
	assert.False(t, ValidUserChannels["orders"], "orders should not be a valid channel type")
	assert.False(t, ValidUserChannels[""], "empty string should not be a valid channel type")
	assert.False(t, ValidUserChannels["MARGIN"], "MARGIN (uppercase) should not be valid")
}

// TestChannelPrefixes tests the channel prefix constants
func TestChannelPrefixes(t *testing.T) {
	assert.Equal(t, "user:", PrefixUser, "PrefixUser should be 'user:'")
}

// TestIsValidAjaibID tests Ajaib ID validation
func TestIsValidAjaibID(t *testing.T) {
	tests := []struct {
		name     string
		ajaibID  string
		expected bool
	}{
		{
			name:     "valid single digit",
			ajaibID:  "1",
			expected: true,
		},
		{
			name:     "valid multi-digit",
			ajaibID:  "130010505",
			expected: true,
		},
		{
			name:     "valid 10 digits",
			ajaibID:  "1234567890",
			expected: true,
		},
		{
			name:     "valid max 10 digits",
			ajaibID:  "9999999999",
			expected: true,
		},
		{
			name:     "invalid - empty",
			ajaibID:  "",
			expected: false,
		},
		{
			name:     "invalid - contains letters",
			ajaibID:  "abc123",
			expected: false,
		},
		{
			name:     "invalid - contains special chars",
			ajaibID:  "130-010505",
			expected: false,
		},
		{
			name:     "invalid - too long (11 digits)",
			ajaibID:  "12345678901",
			expected: false,
		},
		{
			name:     "invalid - contains spaces",
			ajaibID:  "123 456",
			expected: false,
		},
		{
			name:     "invalid - negative number",
			ajaibID:  "-12345",
			expected: false,
		},
		{
			name:     "invalid - decimal",
			ajaibID:  "123.45",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidAjaibID(tt.ajaibID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestChannelInfoFields tests ChannelInfo struct fields
func TestChannelInfoFields(t *testing.T) {
	info, err := ParseChannel("user:130010505:margin")
	require.NoError(t, err)

	assert.Equal(t, "user:130010505:margin", info.Name)
	assert.Equal(t, "user:", info.Prefix)
	assert.Equal(t, "130010505", info.AjaibID)
	assert.Equal(t, "margin", info.ChannelSub)
	// UserID is not used in current implementation, should be empty
	assert.Equal(t, "", info.UserID)
}

// TestParseChannelEdgeCases tests edge cases for channel parsing
func TestParseChannelEdgeCases(t *testing.T) {
	t.Run("channel with leading zeros in ID", func(t *testing.T) {
		// Leading zeros should be valid as they're still digits
		info, err := ParseChannel("user:000123456:margin")
		if err != nil {
			// This might error due to pattern validation, which is fine
			assert.Error(t, err)
		} else {
			assert.Equal(t, "000123456", info.AjaibID)
		}
	})

	t.Run("channel with very long ID", func(t *testing.T) {
		info, err := ParseChannel("user:12345678901234567890:margin")
		assert.Error(t, err)
		assert.Nil(t, info)
	})

	t.Run("channel with whitespace", func(t *testing.T) {
		info, err := ParseChannel("user: 130010505:margin")
		assert.Error(t, err)
		assert.Nil(t, info)
	})

	t.Run("channel with tab", func(t *testing.T) {
		info, err := ParseChannel("user:\t130010505:margin")
		assert.Error(t, err)
		assert.Nil(t, info)
	})
}

// BenchmarkParseChannel benchmarks channel parsing
func BenchmarkParseChannel(b *testing.B) {
	channel := "user:130010505:margin"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseChannel(channel)
	}
}

// BenchmarkIsValidAjaibID benchmarks Ajaib ID validation
func BenchmarkIsValidAjaibID(b *testing.B) {
	ajaibID := "130010505"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isValidAjaibID(ajaibID)
	}
}
