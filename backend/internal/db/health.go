package db 

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)


type SmokeTestResult struct {
	PostgresVersion string
	VendorCount     int
	EntryCount      int
}


func SmokeTest(ctx context.Context, pool *pgxpool.Pool)  (*SmokeTestResult, error) {
	res := &SmokeTestResult{}

	if err := pool.QueryRow(ctx, "SELECT version()").Scan(&res.PostgresVersion); err != nil {
		return nil, fmt.Errorf("SELECT version() failed: %w", err)
	}

	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM vendors").Scan(&res.VendorCount); err != nil {
		return nil, fmt.Errorf("COUNT(*) FROM vendors failed (has the schema been migrated?): %w", err)
	}

	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM vendor_entries").Scan(&res.EntryCount); err != nil {
		return nil, fmt.Errorf("COUNT(*) FROM vendor_entries failed: %w", err)
	}
 
	return res, nil

	
}