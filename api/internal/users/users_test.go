package users_test

import (
	"testing"

	"github.com/hostrix/hostrix/api/internal/users"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := users.HashPassword("secretpass", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !users.CheckPassword(hash, "secretpass") {
		t.Fatal("expected password to match")
	}
	if users.CheckPassword(hash, "wrongpass!") {
		t.Fatal("expected password mismatch")
	}
}

func TestPasswordTooShort(t *testing.T) {
	_, err := users.HashPassword("short", 10)
	if err == nil {
		t.Fatal("expected error for short password")
	}
}
