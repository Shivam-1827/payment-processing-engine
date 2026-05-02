package auth

import (
	"context"
	"crypto/sha256"
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
	hashes := make(map[string]*Merchant)

	// Dynamically compute the real SHA-256 hash of our test key on startup.
	testKey := "sk_live_51M0ckK3y"
	hash := sha256.Sum256([]byte(testKey))
	correctHashHex := hex.EncodeToString(hash[:])

	// Store the dynamically generated hash.
	hashes[correctHashHex] = &Merchant{ID: "merch_001", Name: "Acme Corp"}

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

	// 3. The lookup already proved the hash matches a stored merchant.
	return merchant, nil
}