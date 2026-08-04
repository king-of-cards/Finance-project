package db 

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)


type UserCredentials struct {
	UserID string
	Name  string 
	Role  string 
	PasswordHash string 
}

var ErrUserNotFound = errors.New("user not found ")

func GetUserByID(ctx context.Context, pool *pgxpool.Pool, userID string) (*UserCredentials, error) {
	var u UserCredentials 
	err := pool.QueryRow(ctx, `
		SELECT user_id, name, role, password_hash
		FROM finance_users
		WHERE user_id = $1
	`,userID).Scan(&u.UserID, &u.Name, &u.Role, &u.PasswordHash)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}

	return &u, nil



}