package auth

import (
	"testing"
	"time"
)

func TestPINManager(t *testing.T) {
	sessionToken := "11223344556677889900aabbccddeeff"
	mgr := NewPINManager(sessionToken, 10*time.Minute)

	pin, remaining := mgr.GetActivePIN()
	if len(pin) != 4 {
		t.Fatalf("expected 4-digit PIN string, got length %d (%s)", len(pin), pin)
	}
	if remaining <= 0 {
		t.Fatalf("expected positive remaining duration, got %v", remaining)
	}

	testIP := "192.168.1.55"

	// Test correct PIN
	ok, err := mgr.VerifyPIN(testIP, pin)
	if !ok || err != nil {
		t.Fatalf("expected valid PIN verification, got ok=%v, err=%v", ok, err)
	}

	// Test wrong PIN
	okWrong, errWrong := mgr.VerifyPIN(testIP, "9999_wrong")
	if okWrong || errWrong != ErrInvalidPIN {
		t.Fatalf("expected ErrInvalidPIN on incorrect pin, got: %v", errWrong)
	}

	// Test brute-force lockout after 5 consecutive failures
	lockoutIP := "192.168.1.99"
	for i := 1; i <= 4; i++ {
		ok, err := mgr.VerifyPIN(lockoutIP, "0000_fake")
		if ok || err != ErrInvalidPIN {
			t.Fatalf("attempt %d expected ErrInvalidPIN, got: %v", i, err)
		}
	}

	// 5th failed attempt should trigger lockout
	ok5, err5 := mgr.VerifyPIN(lockoutIP, "0000_fake")
	if ok5 || err5 != ErrIPLockedOut {
		t.Fatalf("5th attempt expected ErrIPLockedOut, got: %v", err5)
	}

	// Subsequent attempt (even with correct PIN) must be rejected during lockout
	okLocked, errLocked := mgr.VerifyPIN(lockoutIP, pin)
	if okLocked || errLocked != ErrIPLockedOut {
		t.Fatalf("locked IP expected ErrIPLockedOut, got: %v", errLocked)
	}

	// Other IP should not be locked out
	otherIP := "192.168.1.100"
	okOther, errOther := mgr.VerifyPIN(otherIP, pin)
	if !okOther || errOther != nil {
		t.Fatalf("other IP should succeed with valid PIN, got: %v", errOther)
	}
}

func TestPINExpiration(t *testing.T) {
	mgr := NewPINManager("token_test", 10*time.Millisecond)
	pin, _ := mgr.GetActivePIN()

	time.Sleep(20 * time.Millisecond)

	ok, err := mgr.VerifyPIN("192.168.1.1", pin)
	if ok || err != ErrPINExpired {
		t.Fatalf("expected ErrPINExpired for expired PIN, got: %v", err)
	}
}
