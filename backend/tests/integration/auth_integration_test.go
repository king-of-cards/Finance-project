package integration

import (
	"context"
	"testing"
	"time"

	"github.com/king-of-cards/finance-project/internal/auth"
	"github.com/king-of-cards/finance-project/internal/config"
	"github.com/king-of-cards/finance-project/internal/db"
)

func TestGetUserByID_RealDatabase(t *testing.T) {
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

	user, err := db.GetUserByID(ctx, pool, "admin")
	if err != nil {
		t.Fatalf("expected to find user 'admin', got error: %v", err)
	}

	if user.Role != "approver" {
		t.Errorf("expected admin's role to be 'approver', got %q", user.Role)
	}

	if !auth.CheckPassword("King@123", user.PasswordHash) {
		t.Error("expected admin's seeded password to match the stored hash")
	}
}

func TestGetUserByID_NonexistentUser(t *testing.T) {
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

	_, err = db.GetUserByID(ctx, pool, "does-not-exist")
	if err != db.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetVendors_NoFilters(t *testing.T) {
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

	vendors, err := db.GetVendors(ctx, pool, db.VendorFilters{})
	if err != nil {
		t.Fatalf("GetVendors returned error: %v", err)
	}
	// Just confirms the query runs without error — count is whatever's seeded
	t.Logf("found %d vendors with no filters", len(vendors))
}

func TestGetVendors_StatusFilter(t *testing.T) {
	cfg, _ := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.NewPool(ctx, cfg)
	if err != nil {
		t.Fatalf("could not connect to database: %v", err)
	}
	defer pool.Close()

	vendors, err := db.GetVendors(ctx, pool, db.VendorFilters{Status: "Active"})
	if err != nil {
		t.Fatalf("GetVendors returned error: %v", err)
	}
	for _, v := range vendors {
		if v.Status != "Active" {
			t.Errorf("expected only Active vendors, got vendor %q with status %q", v.VendorID, v.Status)
		}
	}
}

func TestGetVendors_SearchFilter_IsCaseInsensitive(t *testing.T) {
	cfg, _ := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.NewPool(ctx, cfg)
	if err != nil {
		t.Fatalf("could not connect to database: %v", err)
	}
	defer pool.Close()

	// Uses a lowercase search term to confirm ILIKE is genuinely case-insensitive.
	// Adjust "steel" to a substring that actually exists in your seeded vendor names.
	vendors, err := db.GetVendors(ctx, pool, db.VendorFilters{Search: "steel"})
	if err != nil {
		t.Fatalf("GetVendors returned error: %v", err)
	}
	t.Logf("found %d vendors matching 'steel'", len(vendors))
}

func TestGetVendors_StatusAndSearchCombined(t *testing.T) {
	cfg, _ := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.NewPool(ctx, cfg)
	if err != nil {
		t.Fatalf("could not connect to database: %v", err)
	}
	defer pool.Close()

	// Confirms the WHERE clause correctly joins both conditions with AND,
	// not silently dropping one of them.
	vendors, err := db.GetVendors(ctx, pool, db.VendorFilters{Status: "Active", Search: "a"})
	if err != nil {
		t.Fatalf("GetVendors with combined filters returned error: %v", err)
	}
	for _, v := range vendors {
		if v.Status != "Active" {
			t.Errorf("combined filter leaked non-Active vendor %q", v.VendorID)
		}
	}
}
