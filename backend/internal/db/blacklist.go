package db 

import (
	"context"
	"fmt"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"

)

func RevokeToken(ctx context.Context, pool *pgxpool.Pool, jti string, expiresAt time.Time) error {
	_,err := pool.Exec(ctx,`
		INSERT INTO revoked_tokens (jti, expires_at)
		VALUES ($1, $2)
		ON CONFLICT (jti) DO NOTHING
	`,jti, expiresAt)
	if err != nil {
		return fmt.Errorf("revoking token: %w",err)
	}
	return nil 
}

func IsTokenRevoked(ctx context.Context,pool *pgxpool.Pool, jti string ) (bool, error) {
	var exists bool 
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM revoked_tokens WHERE jti = $1)
	`,jti).Scan(&exists)

	if err!= nil {
		return false, fmt.Errorf("checking revoked token: %w",err)
	}
	return exists, nil 

}