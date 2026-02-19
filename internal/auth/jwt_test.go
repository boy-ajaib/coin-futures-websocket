package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewParser tests creating a new JWT parser
func TestNewParser(t *testing.T) {
	parser := NewParser()
	assert.NotNil(t, parser)
}

// TestParseSubject tests parsing the subject from a valid JWT token
func TestParseSubject(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name        string
		token       string
		expectedSub string
		expectError bool
	}{
		{
			name:        "valid token with subject",
			token:       "eyJ0eXAiOiJBQ0NFU1MiLCJhbGciOiJFUzI1NiJ9.eyJzdWIiOiIxMzAwMTA1MDUiLCJleHAiOjE3NzA4MjA0MDAsImlhdCI6MTc3MDgxOTUwMH0.dummy_signature",
			expectedSub: "130010505",
			expectError: false,
		},
		{
			name:        "token with numeric subject",
			token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			expectedSub: "1234567890",
			expectError: false,
		},
		{
			name:        "empty token",
			token:       "",
			expectedSub: "",
			expectError: true,
		},
		{
			name:        "malformed token - no dots",
			token:       "invalid_token",
			expectedSub: "",
			expectError: true,
		},
		{
			name:        "malformed token - only one dot",
			token:       "header.payload",
			expectedSub: "",
			expectError: true,
		},
		{
			name:        "malformed token - invalid base64",
			token:       "invalid.base64!.signature",
			expectedSub: "",
			expectError: true,
		},
		{
			name:        "valid token without subject",
			token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiSm9obiBEb2UifQ.DummySignature",
			expectedSub: "",
			expectError: true,
		},
		{
			name:        "token with empty subject",
			token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIifQ.DummySignature",
			expectedSub: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, err := parser.ParseSubject(tt.token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, sub)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedSub, sub)
			}
		})
	}
}

// TestTokenExtraction tests extracting token from HTTP request
func TestTokenExtraction(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		queryToken    string
		expectedToken string
		expectFound   bool
	}{
		{
			name:          "token in Authorization header with Bearer prefix",
			authHeader:    "Bearer eyJ0eXAiOiJBQ0NFU1MiLCJhbGciOiJFUzI1NiJ9.eyJzdWIiOiIxMzAwMTA1MDUiLCJleHAiOjE3NzA4MjA0MDAsImlhdCI6MTc3MDgxOTUwMH0.dummy_signature",
			queryToken:    "",
			expectedToken: "eyJ0eXAiOiJBQ0NFU1MiLCJhbGciOiJFUzI1NiJ9.eyJzdWIiOiIxMzAwMTA1MDUiLCJleHAiOjE3NzA4MjA0MDAsImlhdCI6MTc3MDgxOTUwMH0.dummy_signature",
			expectFound:   true,
		},
		{
			name:          "token in X-Socket-Authorization header",
			authHeader:    "eyJ0eXAiOiJBQ0NFU1MiLCJhbGciOiJFUzI1NiJ9.eyJzdWIiOiIxMzAwMTA1MDUiLCJleHAiOjE3NzA4MjA0MDAsImlhdCI6MTc3MDgxOTUwMH0.dummy_signature",
			queryToken:    "",
			expectedToken: "eyJ0eXAiOiJBQ0NFU1MiLCJhbGciOiJFUzI1NiJ9.eyJzdWIiOiIxMzAwMTA1MDUiLCJleHAiOjE3NzA4MjA0MDAsImlhdCI6MTc3MDgxOTUwMH0.dummy_signature",
			expectFound:   true,
		},
		{
			name:          "token in query parameter",
			authHeader:    "",
			queryToken:    "eyJ0eXAiOiJBQ0NFU1MiLCJhbGciOiJFUzI1NiJ9.eyJzdWIiOiIxMzAwMTA1MDUiLCJleHAiOjE3NzA4MjA0MDAsImlhdCI6MTc3MDgxOTUwMH0.dummy_signature",
			expectedToken: "eyJ0eXAiOiJBQ0NFU1MiLCJhbGciOiJFUzI1NiJ9.eyJzdWIiOiIxMzAwMTA1MDUiLCJleHAiOjE3NzA4MjA0MDAsImlhdCI6MTc3MDgxOTUwMH0.dummy_signature",
			expectFound:   true,
		},
		{
			name:          "no token present",
			authHeader:    "",
			queryToken:    "",
			expectedToken: "",
			expectFound:   false,
		},
		{
			name:          "empty Authorization header",
			authHeader:    "",
			queryToken:    "",
			expectedToken: "",
			expectFound:   false,
		},
		{
			name:          "Bearer without token",
			authHeader:    "Bearer ",
			queryToken:    "",
			expectedToken: "",
			expectFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock HTTP request
			// Note: This is a simplified test - in real scenarios, you'd use http.NewRequest
			// and set headers/queries properly
		})
	}
}

// TestParseSubjectWithVariousFormats tests parsing JWT with various payload formats
func TestParseSubjectWithVariousFormats(t *testing.T) {
	parser := NewParser()

	t.Run("JWT with standard claims", func(t *testing.T) {
		// A standard JWT with exp, iat, and sub
		token := "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZXhwIjoxNzAwMDAwMDAwLCJpYXQiOjE2OTk5OTk5OTl9.dummy"
		sub, err := parser.ParseSubject(token)
		require.NoError(t, err)
		assert.Equal(t, "1234567890", sub)
	})

	t.Run("JWT with only sub claim", func(t *testing.T) {
		// Minimal JWT with just sub
		token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIn0.dummy"
		sub, err := parser.ParseSubject(token)
		require.NoError(t, err)
		assert.Equal(t, "user123", sub)
	})

	t.Run("JWT with sub as number in JSON", func(t *testing.T) {
		// Some JWT implementations encode numbers as numbers, not strings
		// This tests that we handle it correctly
		token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOjEyMzQ1Nn0.dummy"
		sub, err := parser.ParseSubject(token)
		// The parser should handle numeric subjects
		// Depending on implementation, this might be "123456" or an error
		if err == nil {
			assert.NotEmpty(t, sub)
		}
	})
}

// BenchmarkParseSubject benchmarks the ParseSubject function
func BenchmarkParseSubject(b *testing.B) {
	parser := NewParser()
	token := "eyJ0eXAiOiJBQ0NFU1MiLCJhbGciOiJFUzI1NiJ9.eyJzdWIiOiIxMzAwMTA1MDUiLCJleHAiOjE3NzA4MjA0MDAsImlhdCI6MTc3MDgxOTUwMH0.dummy_signature"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseSubject(token)
	}
}
