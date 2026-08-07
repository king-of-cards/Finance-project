package smoke

import (
	"context"
	"testing"
	"time"

	"github.com/king-of-cards/finance-project/internal/config"
	"github.com/king-of-cards/finance-project/internal/db"
)

func TestDatabaseIsReachable(t *testing.T) {
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

	var result int
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)

	if err != nil {
		t.Fatalf("smoke query failed: %v", err)
	}
	if result != 1 {
		t.Fatalf("expected 1, got %d", result)
	}

}

// TestFinanceUsersTableExists confirms the core table your whole app depends on is reachable.
func TestFinanceUsersTableExists(t *testing.T) {
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

	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM finance_users").Scan(&count)
	if err != nil {
		t.Fatalf("finance_users table check failed: %v", err)
	}
	if count == 0 {
		t.Error("expected at least the 3 seeded users (admin, manoj, firdosh), got 0 rows")
	}
}
