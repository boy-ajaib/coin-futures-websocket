package crypto

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// OpenSSH private key format constants
const (
	opensshPrivateKeyMagic = "openssh-key-v1\x00"
	ed25519KeyType         = "ssh-ed25519"
)

// Ed25519KeyPair holds parsed Ed25519 keys
type Ed25519KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// LoadEd25519KeyFromFile loads an Ed25519 private key from an OpenSSH format file
func LoadEd25519KeyFromFile(path string) (*Ed25519KeyPair, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	return ParseOpenSSHPrivateKey(data)
}

// ParseOpenSSHPrivateKey parses an OpenSSH format Ed25519 private key
func ParseOpenSSHPrivateKey(pemData []byte) (*Ed25519KeyPair, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	if block.Type != "OPENSSH PRIVATE KEY" {
		return nil, fmt.Errorf("unexpected key type: %s", block.Type)
	}

	return parseOpenSSHPrivateKeyBytes(block.Bytes)
}

// parseOpenSSHPrivateKeyBytes parses the raw OpenSSH private key format
// Format: https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.key
func parseOpenSSHPrivateKeyBytes(data []byte) (*Ed25519KeyPair, error) {
	// Check magic header
	if len(data) < len(opensshPrivateKeyMagic) {
		return nil, errors.New("key data too short")
	}

	if string(data[:len(opensshPrivateKeyMagic)]) != opensshPrivateKeyMagic {
		return nil, errors.New("invalid openssh key magic")
	}

	pos := len(opensshPrivateKeyMagic)

	// Read cipher name (should be "none" for unencrypted)
	cipherName, newPos, err := readString(data, pos)
	if err != nil {
		return nil, fmt.Errorf("failed to read cipher name: %w", err)
	}
	pos = newPos

	if cipherName != "none" {
		return nil, fmt.Errorf("encrypted keys not supported, cipher: %s", cipherName)
	}

	// Read KDF name (should be "none" for unencrypted)
	kdfName, pos, err := readString(data, pos)
	if err != nil {
		return nil, fmt.Errorf("failed to read KDF name: %w", err)
	}

	if kdfName != "none" {
		return nil, fmt.Errorf("encrypted keys not supported, kdf: %s", kdfName)
	}

	// Read KDF options (empty for unencrypted)
	_, pos, err = readString(data, pos)
	if err != nil {
		return nil, fmt.Errorf("failed to read KDF options: %w", err)
	}

	// Read number of keys (should be 1)
	if pos+4 > len(data) {
		return nil, errors.New("key data truncated at key count")
	}
	numKeys := readUint32(data, pos)
	pos += 4

	if numKeys != 1 {
		return nil, fmt.Errorf("expected 1 key, got %d", numKeys)
	}

	// Read public key blob (skip it, we'll get it from private section)
	_, pos, err = readString(data, pos)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key blob: %w", err)
	}

	// Read private key section
	privateSection, _, err := readString(data, pos)
	if err != nil {
		return nil, fmt.Errorf("failed to read private section: %w", err)
	}

	return parsePrivateSection([]byte(privateSection))
}

// parsePrivateSection parses the private key section of an OpenSSH key
func parsePrivateSection(data []byte) (*Ed25519KeyPair, error) {
	if len(data) < 8 {
		return nil, errors.New("private section too short")
	}

	// Read check integers (should match for unencrypted keys)
	check1 := readUint32(data, 0)
	check2 := readUint32(data, 4)

	if check1 != check2 {
		return nil, errors.New("check integers don't match (possibly encrypted)")
	}

	pos := 8

	// Read key type
	keyType, pos, err := readString(data, pos)
	if err != nil {
		return nil, fmt.Errorf("failed to read key type: %w", err)
	}

	if keyType != ed25519KeyType {
		return nil, fmt.Errorf("expected ed25519 key, got: %s", keyType)
	}

	// Read public key (32 bytes)
	pubKeyStr, pos, err := readString(data, pos)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	if len(pubKeyStr) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: %d", len(pubKeyStr))
	}

	// Read private key (64 bytes: 32 seed + 32 public)
	privKeyStr, _, err := readString(data, pos)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	if len(privKeyStr) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: %d", len(privKeyStr))
	}

	privateKey := ed25519.PrivateKey([]byte(privKeyStr))
	publicKey := ed25519.PublicKey([]byte(pubKeyStr))

	return &Ed25519KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// Sign signs a message using the private key and returns base64-encoded signature
func (kp *Ed25519KeyPair) Sign(message []byte) string {
	signature := ed25519.Sign(kp.PrivateKey, message)
	return base64.StdEncoding.EncodeToString(signature)
}

// readString reads a length-prefixed string from data at position
func readString(data []byte, pos int) (string, int, error) {
	if pos+4 > len(data) {
		return "", 0, errors.New("data truncated reading string length")
	}

	length := int(readUint32(data, pos))
	pos += 4

	if pos+length > len(data) {
		return "", 0, errors.New("data truncated reading string content")
	}

	return string(data[pos : pos+length]), pos + length, nil
}

// readUint32 reads a big-endian uint32 from data at position
func readUint32(data []byte, pos int) uint32 {
	return uint32(data[pos])<<24 |
		uint32(data[pos+1])<<16 |
		uint32(data[pos+2])<<8 |
		uint32(data[pos+3])
}
