package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

// LoadPrivateKeyFromHex loads an ECDSA private key from hex string
func LoadPrivateKeyFromHex(hexKey string) (*ecdsa.PrivateKey, error) {
	// Remove 0x prefix
	hexKey = strings.TrimPrefix(hexKey, "0x")

	// Decode hex string
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex key: %v", err)
	}

	// Create private key
	privateKey := new(ecdsa.PrivateKey)
	privateKey.Curve = elliptic.P256()
	privateKey.D = new(big.Int).SetBytes(keyBytes)

	// Calculate public key
	privateKey.PublicKey.X, privateKey.PublicKey.Y = privateKey.Curve.ScalarBaseMult(keyBytes)

	return privateKey, nil
}

// GeneratePrivateKey generates a new ECDSA private key
func GeneratePrivateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// PrivateKeyToHex converts a private key to hex string
func PrivateKeyToHex(privateKey *ecdsa.PrivateKey) string {
	keyBytes := privateKey.D.Bytes()
	return hex.EncodeToString(keyBytes)
}

// PublicKeyToHex converts a public key to hex string
func PublicKeyToHex(publicKey *ecdsa.PublicKey) string {
	x := publicKey.X.Bytes()
	y := publicKey.Y.Bytes()

	// Pad to 32 bytes
	for len(x) < 32 {
		x = append([]byte{0}, x...)
	}
	for len(y) < 32 {
		y = append([]byte{0}, y...)
	}

	return hex.EncodeToString(append(x, y...))
}

// SignData signs data with ECDSA private key
func SignData(privateKey *ecdsa.PrivateKey, data []byte) (string, error) {
	// Calculate data hash
	hash := sha256.Sum256(data)

	// Sign
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign data: %v", err)
	}

	// Encode r and s as hex strings
	signature := append(r.Bytes(), s.Bytes()...)
	return hex.EncodeToString(signature), nil
}

// VerifySignature verifies ECDSA signature
func VerifySignature(publicKey *ecdsa.PublicKey, data []byte, signature string) (bool, error) {
	// Decode signature
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %v", err)
	}

	if len(sigBytes) != 64 {
		return false, fmt.Errorf("invalid signature length: %d", len(sigBytes))
	}

	// Extract r and s
	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:])

	// Calculate data hash
	hash := sha256.Sum256(data)

	// Verify signature
	valid := ecdsa.Verify(publicKey, hash[:], r, s)
	return valid, nil
}

// HashData computes SHA256 hash of data
func HashData(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// HashDataHex computes SHA256 hash and returns hex string
func HashDataHex(data []byte) string {
	hash := HashData(data)
	return hex.EncodeToString(hash)
}
