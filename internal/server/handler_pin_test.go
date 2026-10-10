package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"warpdrop/internal/auth"
)

func TestPINAuthHandler(t *testing.T) {
	sessionToken := "11223344556677889900aabbccddeeff"
	pinMgr := auth.NewPINManager(sessionToken, 10*time.Minute)
	handler := PINAuthHandler(pinMgr)

	activePIN, _ := pinMgr.GetActivePIN()

	// Case 1: Valid PIN submission
	reqBody, _ := json.Marshal(map[string]string{"pin": activePIN})
	req := httptest.NewRequest("POST", "/api/v1/auth/pin", bytes.NewReader(reqBody))
	req.RemoteAddr = "192.168.1.50:51234"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got: %d (%s)", rec.Code, rec.Body.String())
	}

	var resp pinAuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if resp.Status != "success" || resp.Token != sessionToken {
		t.Fatalf("expected success with sessionToken, got: %+v", resp)
	}

	// Verify session cookie was set
	cookies := rec.Result().Cookies()
	var hasCookie bool
	for _, c := range cookies {
		if c.Name == sessionCookieName && c.Value == sessionToken {
			hasCookie = true
			break
		}
	}
	if !hasCookie {
		t.Fatal("expected session cookie to be set on valid PIN authentication")
	}

	// Case 2: Invalid PIN submission
	reqBad, _ := json.Marshal(map[string]string{"pin": "0000_fake"})
	reqBadHTTP := httptest.NewRequest("POST", "/api/v1/auth/pin", bytes.NewReader(reqBad))
	reqBadHTTP.RemoteAddr = "192.168.1.60:51234"
	recBad := httptest.NewRecorder()

	handler.ServeHTTP(recBad, reqBadHTTP)

	if recBad.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized for wrong PIN, got: %d", recBad.Code)
	}

	// Case 3: Brute force lockout (5 failed attempts from same IP)
	lockoutIP := "192.168.1.77:50000"
	for i := 1; i <= 4; i++ {
		reqF, _ := json.Marshal(map[string]string{"pin": "0000"})
		r := httptest.NewRequest("POST", "/api/v1/auth/pin", bytes.NewReader(reqF))
		r.RemoteAddr = lockoutIP
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d expected 401, got %d", i, w.Code)
		}
	}

	// 5th attempt triggers 429 Too Many Requests
	req5, _ := json.Marshal(map[string]string{"pin": "0000"})
	r5 := httptest.NewRequest("POST", "/api/v1/auth/pin", bytes.NewReader(req5))
	r5.RemoteAddr = lockoutIP
	w5 := httptest.NewRecorder()
	handler.ServeHTTP(w5, r5)

	if w5.Code != http.StatusTooManyRequests {
		t.Fatalf("5th attempt expected status 429 Too Many Requests, got: %d", w5.Code)
	}
}
