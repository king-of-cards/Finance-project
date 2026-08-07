package auth

import "testing"

func TestHashPassword_ProducesDifferentHashForSamePassword(t *testing.T) {
	hash1, err := HashPassword("King@123")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	hash2, err := HashPassword("King@123")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash1 == hash2 {
		t.Error("expected two hashes of the same password to differ (bcrypt salts them), but they matched")
	}
}

func TestCheckPassword_CorrectPassword(t *testing.T) {
	hash, _ := HashPassword("King@123")
	if !CheckPassword("King@123", hash) {
		t.Error("expected CheckPassword to return true for the correct password")
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, _ := HashPassword("King@123")
	if CheckPassword("WrongPassword", hash) {
		t.Error("expected CheckPassword to return false for an incorrect password")
	}
}

func TestCheckPassword_EmptyHash(t *testing.T) {
	if CheckPassword("anything", "") {
		t.Error("expected CheckPassword to return false when hash is empty")
	}
}