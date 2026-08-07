package e2e

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/king-of-cards/finance-project/internal/config"
	"github.com/king-of-cards/finance-project/internal/db"
	"github.com/king-of-cards/finance-project/internal/router"
)

func TestFullLoginLogoutFlow(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg)
	if err != nil {
		t.Fatalf("could not connect to database: %v", err)
	}
	defer pool.Close()

	r := router.New(pool, cfg.JWTSecret, cfg.JWTExpiryHours)

	// Step 1: Login
	loginBody := `{"user_id":"manoj","password":"Manoj@123"}`
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("login failed: %d, %s", w.Code, w.Body.String())
	}

	var loginResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	token := loginResp["token"].(string)

	// Step 2: Access protected route with token — should succeed
	req2 := httptest.NewRequest("GET", "/api/auth/me", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != 200 {
		t.Fatalf("expected /auth/me to succeed with valid token, got %d: %s", w2.Code, w2.Body.String())
	}

	// Step 3: Logout
	req3 := httptest.NewRequest("POST", "/api/logout", nil)
	req3.Header.Set("Authorization", "Bearer "+token)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	if w3.Code != 200 {
		t.Fatalf("logout failed: %d, %s", w3.Code, w3.Body.String())
	}

	// Step 4: Reuse same token — should now be rejected
	req4 := httptest.NewRequest("GET", "/api/auth/me", nil)
	req4.Header.Set("Authorization", "Bearer "+token)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)

	if w4.Code != 401 {
		t.Fatalf("expected 401 for revoked token, got %d: %s", w4.Code, w4.Body.String())
	}
}
