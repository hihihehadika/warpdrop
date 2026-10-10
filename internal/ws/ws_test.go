package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHubBroadcastAndEchoPrevention(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := r.URL.Query().Get("id")
		ServeWs(hub, w, r, clientID)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect Client 1
	dialer := websocket.DefaultDialer
	conn1, _, err := dialer.Dial(wsURL+"?id=client_1", nil)
	if err != nil {
		t.Fatalf("failed to dial client 1: %v", err)
	}
	defer conn1.Close()

	// Connect Client 2
	conn2, _, err := dialer.Dial(wsURL+"?id=client_2", nil)
	if err != nil {
		t.Fatalf("failed to dial client 2: %v", err)
	}
	defer conn2.Close()

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != 2 {
		t.Fatalf("expected 2 connected clients, got: %d", hub.ClientCount())
	}

	// Client 1 sends clipboard text
	testMessage := Message{
		Type:    "clipboard",
		Content: "https://github.com/hihihehadika/warpdrop",
	}

	if err := conn1.WriteJSON(testMessage); err != nil {
		t.Fatalf("client 1 failed to send json: %v", err)
	}

	// Client 2 should receive the clipboard message
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	var received Message
	if err := conn2.ReadJSON(&received); err != nil {
		t.Fatalf("client 2 failed to read broadcasted message: %v", err)
	}

	if received.Type != "clipboard" || received.Content != testMessage.Content {
		t.Fatalf("unexpected message content received: %+v", received)
	}
	if received.SenderID != "client_1" {
		t.Fatalf("expected SenderID client_1, got: %s", received.SenderID)
	}
}
