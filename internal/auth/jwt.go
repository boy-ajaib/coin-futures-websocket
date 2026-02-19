package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Parser handles JWT parsing and claim extraction.
// This is a minimal JWT parser that only decodes the payload without signature verification.
// Signature verification should be handled by the authentication service before the token reaches this service.
type Parser struct{}

// NewParser creates a new JWT parser.
func NewParser() *Parser {
	return &Parser{}
}

// Claims represents the standard JWT claims we need.
type Claims struct {
	Sub string `json:"sub"` // Subject - user identifier
}

// Parse extracts the subject (sub) claim from a JWT token.
func (p *Parser) Parse(token string) (*Claims, error) {
	if token == "" {
		return nil, fmt.Errorf("empty token")
	}

	// Remove Bearer prefix if present
	token = trimBearerPrefix(token)

	parts, err := splitToken(token)
	if err != nil {
		return nil, err
	}

	payload, err := decodePayload(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	claims, err := unmarshalClaims(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	if claims.Sub == "" {
		return nil, fmt.Errorf("empty sub in JWT claims")
	}

	return claims, nil
}

// ParseSubject extracts just the subject from a JWT token.
func (p *Parser) ParseSubject(token string) (string, error) {
	claims, err := p.Parse(token)
	if err != nil {
		return "", err
	}
	return claims.Sub, nil
}

// trimBearerPrefix removes "Bearer " prefix from token if present.
func trimBearerPrefix(token string) string {
	return trimPrefix(token, "Bearer ")
}

// trimPrefix removes a prefix from a string.
func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

// splitToken splits a JWT token into its 3 parts.
func splitToken(token string) ([]string, error) {
	parts := splitString(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}
	return parts, nil
}

// splitString is a simple strings.Split implementation.
func splitString(s, sep string) []string {
	if sep == "" {
		return []string{s}
	}
	n := countSubstring(s, sep) + 1
	result := make([]string, n)
	i := 0
	for {
		m := indexString(s, sep)
		if m < 0 {
			break
		}
		result[i] = s[:m]
		i++
		s = s[m+len(sep):]
	}
	result[i] = s
	return result
}

// countSubstring counts non-overlapping occurrences of sep in s.
func countSubstring(s, sep string) int {
	count := 0
	for {
		i := indexString(s, sep)
		if i < 0 {
			break
		}
		count++
		s = s[i+len(sep):]
	}
	return count
}

// indexString returns the index of the first occurrence of sep in s, or -1 if not found.
func indexString(s, sep string) int {
	n := len(sep)
	if n == 0 {
		return 0
	}
	if n > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-n; i++ {
		if s[i:i+n] == sep {
			return i
		}
	}
	return -1
}

// decodePayload base64 decodes a JWT payload.
func decodePayload(payload string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(payload)
}

// unmarshalClaims unmarshals JWT claims from JSON.
func unmarshalClaims(data []byte) (*Claims, error) {
	var claims Claims
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, err
	}
	return &claims, nil
}

// ExtractTokenFromRequest extracts a JWT token from an HTTP request.
// It checks the Authorization header first, then falls back to the token query parameter.
type TokenExtractor struct{}

// NewTokenExtractor creates a new token extractor.
func NewTokenExtractor() *TokenExtractor {
	return &TokenExtractor{}
}

// Extract extracts a JWT token from an HTTP request.
func (e *TokenExtractor) Extract(authorizationHeader, tokenQueryParam string) (string, error) {
	// Try Authorization header first
	if authorizationHeader != "" {
		token := trimBearerPrefix(authorizationHeader)
		if token != "" {
			return token, nil
		}
	}

	// Fallback to query parameter
	if tokenQueryParam != "" {
		return tokenQueryParam, nil
	}

	return "", fmt.Errorf("no token found in Authorization header or token query parameter")
}
