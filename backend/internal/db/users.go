package db 

import (
	"context"
	"fmt"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"

)

type FinanceUser struct {
	UserID string `json:"user_id"`
	Name string `json:"name"`
	Role string  `json:"role"`
	Email *string `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func GetAllUsers(ctx context.Context, pool *pgxpool.Pool) ([]FinanceUser,error) {
	rows, err := pool.Query(ctx, `
		SELECT user_id, name, role, email, created_at
		FROM finance_users
		ORDER BY name
	`)

	if err!= nil {
		return nil, fmt.Errorf("quering finance_users: %w",err)
	}
	defer rows.Close()

	var users []FinanceUser
	for rows.Next() {
		var u FinanceUser
		if err := rows.Scan(&u.UserID, &u.Name, &u.Role, &u.Email, &u.CreatedAt);
		err != nil {
			return nil, fmt.Errorf("scanning finance_user row: %w",err)
		}
		users = append(users,u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating finance_user rows:%w",err)
	}
	return users, nil 
}

