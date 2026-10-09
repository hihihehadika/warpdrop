package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	expectedToken := "11223344556677889900aabbccddeeff"
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authorized"))
	})

	handler := AuthMiddleware(expectedToken, dummyHandler)

	// Case 1: Unauthorized request (no token)
	req := httptest.NewRequest("GET", "/api/v1/files", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got: %d", rec.Code)
	}

	// Case 2: Unauthorized request (tampered token)
	reqTampered := httptest.NewRequest("GET", "/api/v1/files?token=wrong_token", nil)
	recTampered := httptest.NewRecorder()
	handler.ServeHTTP(recTampered, reqTampered)

	if recTampered.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden on wrong token, got: %d", recTampered.Code)
	}

	// Case 3: Authorized via query parameter
	reqQuery := httptest.NewRequest("GET", "/api/v1/files?token="+expectedToken, nil)
	recQuery := httptest.NewRecorder()
	handler.ServeHTTP(recQuery, reqQuery)

	if recQuery.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK with query token, got: %d", recQuery.Code)
	}

	// Verify session cookie was set
	cookies := recQuery.Result().Cookies()
	var foundCookie bool
	for _, c := range cookies {
		if c.Name == sessionCookieName && c.Value == expectedToken {
			foundCookie = true
			break
		}
	}
	if !foundCookie {
		t.Fatal("expected session cookie to be set on successful authentication")
	}

	// Case 4: Authorized via HTTP Header
	reqHeader := httptest.NewRequest("GET", "/api/v1/files", nil)
	reqHeader.Header.Set("X-WarpDrop-Token", expectedToken)
	recHeader := httptest.NewRecorder()
	handler.ServeHTTP(recHeader, reqHeader)

	if recHeader.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK with header token, got: %d", recHeader.Code)
	}

	// Case 5: Authorized via Cookie
	reqCookie := httptest.NewRequest("GET", "/api/v1/files", nil)
	reqCookie.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: expectedToken,
	})
	recCookie := httptest.NewRecorder()
	handler.ServeHTTP(recCookie, reqCookie)

	if recCookie.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK with session cookie, got: %d", recCookie.Code)
	}
}
