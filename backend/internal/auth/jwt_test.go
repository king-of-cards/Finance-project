package auth

import "testing"

const testSecret = "test-secret-for-unit-tests-only"

func TestGenerateToken_SetsJTI(t *testing.T) {
	tokenStr, _, jti, err := GenerateToken("admin", "approver", testSecret, 1)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	if jti == "" {
		t.Fatal("expected GenerateToken to return a non-empty jti")
	}

	claims, err := ValidateToken(tokenStr, testSecret)
	if err != nil {
		t.Fatalf("ValidateToken returned error: %v", err)
	}
	if claims.ID != jti {
		t.Errorf("expected claims.ID to equal returned jti %q, got %q", jti, claims.ID)
	}
}

func TestGenerateAndValidateToken_RoundTrip(t *testing.T) {
	tokenStr, _, _, err := GenerateToken("manoj", "finance", testSecret, 1)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	claims, err := ValidateToken(tokenStr, testSecret)
	if err != nil {
		t.Fatalf("ValidateToken returned error: %v", err)
	}
	if claims.UserID != "manoj" {
		t.Errorf("expected UserID 'manoj', got %q", claims.UserID)
	}
	if claims.Role != "finance" {
		t.Errorf("expected Role 'finance', got %q", claims.Role)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	tokenStr, _, _, err := GenerateToken("admin", "approver", testSecret, 1)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	_, err = ValidateToken(tokenStr, "a-completely-different-secret")
	if err == nil {
		t.Error("expected ValidateToken to fail with the wrong secret, but it succeeded")
	}
}

func TestValidateToken_GarbageInput(t *testing.T) {
	_, err := ValidateToken("not.a.real.jwt", testSecret)
	if err == nil {
		t.Error("expected ValidateToken to fail on garbage input, but it succeeded")
	}
}
