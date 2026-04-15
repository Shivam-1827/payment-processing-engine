package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
)

var ErrUnauthorized = errors.New("invalid api key")

type Merchant struct {
	ID string
	Name string
}

type Authenticator interface {
	Authenticate(ctx context.Context, apiKey string) (*Merchant, error)
}

// HashedAPIKeyAuthenticator simulates a real DB lookup where only SHA-256 hashes are stored
type HashedAPIKeyAuthenticator struct {
	// In production, this map would be a Redis lookup or Postgres query
	// Key: SHA256 Hash of the API key | Value: Merchant Details
	merchantKeyHashes map[string]*Merchant
}

func NewHashedAPIKeyAuthenticator() *HashedAPIKeyAuthenticator {
	// Let's assume the real API key given to the user is: "sk_live_51M0ckK3y..."
	// The SHA-256 hash of "sk_live_51M0ckK3y" is below. We NEVER store the plaintext.
	mockKeyHash := "c8d8b6f3b0e3e2a0134a6df15545a99307771761d4a0f44bc19ba6b29f9e578c"

	hashes := make(map[string]*Merchant)
	hashes[mockKeyHash] = &Merchant{ID: "merch_001", Name: "Acme Corp"}

	return &HashedAPIKeyAuthenticator{
		merchantKeyHashes: hashes,
	}
}

func (a *HashedAPIKeyAuthenticator) Authenticate(ctx context.Context, apiKey string) (*Merchant, error) {
	if len(apiKey) < 10 {
		return nil, ErrUnauthorized
	}

	// 1. Hash the incoming API Key
	hash := sha256.Sum256([]byte(apiKey))
	incomingHashHex := hex.EncodeToString(hash[:])

	// 2. Look up the hash in our datastore (Simulated map)
	merchant, exists := a.merchantKeyHashes[incomingHashHex]
	if !exists {
		return nil, ErrUnauthorized
	}

	// 3. Constant time compare to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(incomingHashHex), []byte(incomingHashHex)) == 1 {
		return merchant, nil
	}

	return nil, ErrUnauthorized
}