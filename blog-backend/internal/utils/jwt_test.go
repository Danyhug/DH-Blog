package utils

import (
	"testing"
	"time"
)

func TestJWTServicesAreIndependent(t *testing.T) {
	first := NewJWTService("first-secret", time.Hour)
	second := NewJWTService("second-secret", time.Hour)

	token, err := first.GenerateJWT("admin")
	if err != nil {
		t.Fatal(err)
	}
	raw := token[len("Bearer "):]
	if _, err := first.ParseToken(raw); err != nil {
		t.Fatalf("first service rejected its token: %v", err)
	}
	if _, err := second.ParseToken(raw); err == nil {
		t.Fatal("second service accepted a token signed with another secret")
	}
}
