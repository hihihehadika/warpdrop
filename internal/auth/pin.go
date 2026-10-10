package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

var (
	// ErrIPLockedOut returned when client IP is in lockout cooldown due to repeated failures.
	ErrIPLockedOut = errors.New("terlalu banyak percobaan gagal, silakan tunggu 60 detik")
	// ErrPINExpired returned when current PIN has passed its expiration time.
	ErrPINExpired = errors.New("kode PIN telah kedaluwarsa")
	// ErrInvalidPIN returned when submitted PIN does not match active PIN.
	ErrInvalidPIN = errors.New("kode PIN salah")
)

type ipRecord struct {
	failedAttempts int
	lockedUntil    time.Time
}

// PINManager manages ephemeral 4-digit quick pairing PINs, expiration, and rate-limited brute-force lockouts.
type PINManager struct {
	mu           sync.RWMutex
	activePIN    string
	sessionToken string
	expiresAt    time.Time
	ttl          time.Duration
	maxAttempts  int
	lockDuration time.Duration
	ipRecords    map[string]*ipRecord
}

// NewPINManager initializes a PINManager instance with given session token and expiration duration.
func NewPINManager(sessionToken string, ttl time.Duration) *PINManager {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}

	mgr := &PINManager{
		sessionToken: sessionToken,
		ttl:          ttl,
		maxAttempts:  5,
		lockDuration: 60 * time.Second,
		ipRecords:    make(map[string]*ipRecord),
	}

	mgr.RegeneratePIN()
	return mgr
}

// RegeneratePIN generates a new random 4-digit PIN between 0000 and 9999.
func (m *PINManager) RegeneratePIN() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		m.activePIN = "7419" // Fallback deterministic pin if rand entropy error
	} else {
		m.activePIN = fmt.Sprintf("%04d", n.Int64())
	}

	m.expiresAt = time.Now().Add(m.ttl)
	return m.activePIN
}

// GetActivePIN returns the current PIN and remaining validity duration.
func (m *PINManager) GetActivePIN() (string, time.Duration) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	remaining := time.Until(m.expiresAt)
	if remaining < 0 {
		remaining = 0
	}
	return m.activePIN, remaining
}

// GetSessionToken returns the underlying full-length cryptographic session token.
func (m *PINManager) GetSessionToken() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessionToken
}

// VerifyPIN validates client-supplied PIN against the active PIN with constant-time comparison and lockout protection.
func (m *PINManager) VerifyPIN(clientIP, pin string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Check IP lockout record
	rec, exists := m.ipRecords[clientIP]
	if !exists {
		rec = &ipRecord{}
		m.ipRecords[clientIP] = rec
	}

	if now.Before(rec.lockedUntil) {
		return false, ErrIPLockedOut
	}

	// Check expiration
	if now.After(m.expiresAt) {
		return false, ErrPINExpired
	}

	// Constant-time PIN comparison
	match := subtle.ConstantTimeCompare([]byte(pin), []byte(m.activePIN)) == 1
	if match {
		rec.failedAttempts = 0
		rec.lockedUntil = time.Time{}
		return true, nil
	}

	// Record failed attempt
	rec.failedAttempts++
	if rec.failedAttempts >= m.maxAttempts {
		rec.lockedUntil = now.Add(m.lockDuration)
		return false, ErrIPLockedOut
	}

	return false, ErrInvalidPIN
}
