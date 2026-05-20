package ws

import "testing"

func TestIsClientMessageTypeAllowed(t *testing.T) {
	allowed := []WSMessageType{WSMChat, WSMTyping, WSMReadReceipt}
	for _, messageType := range allowed {
		if !IsClientMessageTypeAllowed(messageType) {
			t.Fatalf("expected %s to be allowed", messageType)
		}
	}

	blocked := []WSMessageType{WSMStatus, WSMAck, WSMError, "admin"}
	for _, messageType := range blocked {
		if IsClientMessageTypeAllowed(messageType) {
			t.Fatalf("expected %s to be blocked", messageType)
		}
	}
}
