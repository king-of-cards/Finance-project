package main 

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/king-of-cards/finance-project/internal/auth"
	"github.com/king-of-cards/finance-project/internal/config"
	"github.com/king-of-cards/finance-project/internal/db"
)

type seedUser struct {
	UserID   string
	Name     string
	Role     string
	Email    string
	Password string
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg)
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	defer pool.Close()

	users := []seedUser{
	{UserID: "admin", Name: "Admin", Role: "approver", Email: "admin@kingofcards.com", Password: "King@123"},
	{UserID: "manoj", Name: "Manoj", Role: "finance", Email: "manoj@kingofcards.com", Password: "Manoj@123"},
	{UserID: "firdosh", Name: "Firdosh", Role: "finance", Email: "firdosh@kingofcards.com", Password: "Firdosh@123"},
	}

	for _, u := range users {
		hash, err := auth.HashPassword(u.Password)
		if err != nil {
			log.Fatalf("hashing password for %s: %v", u.UserID, err)
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO finance_users (user_id, name, role, email, password_hash)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (user_id)
			DO UPDATE SET password_hash = EXCLUDED.password_hash, name = EXCLUDED.name, role = EXCLUDED.role
		`, u.UserID, u.Name, u.Role, u.Email, hash)

		if err != nil {
			log.Fatalf("seeding user %s: %v", u.UserID, err)
		}

		fmt.Printf("✅ seeded user: %s (role: %s)\n", u.UserID, u.Role)
	}

	fmt.Println("done.")


}