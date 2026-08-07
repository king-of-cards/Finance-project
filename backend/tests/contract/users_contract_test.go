package contract

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

func TestLoginResponse_HasExpectedShape(t *testing.T) {
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

	body := `{"user_id":"admin","password":"King@123"}`
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	requiredFields := []string{"token", "expires_at", "user"}
	for _, field := range requiredFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("expected field %q in login response, missing", field)
		}
	}

	user, ok := resp["user"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'user' field to be an object")
	}
	for _, field := range []string{"user_id", "name", "role"} {
		if _, ok := user[field]; !ok {
			t.Errorf("expected field %q inside 'user', missing", field)
		}
	}
}
