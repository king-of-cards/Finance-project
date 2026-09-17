package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Comment struct {
	CommentID   int64     `json:"comment_id"`
	PONumber    string    `json:"po_number"`
	AuthorID    string    `json:"author_id"`
	AuthorName  string    `json:"author_name"`
	CommentText string    `json:"comment_text"`
	CreatedAt   time.Time `json:"created_at"`
}

var ErrCommentNotFound = errors.New("comment not found")

func GetComments(ctx context.Context, pool *pgxpool.Pool, poNumber string) ([]Comment, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM finance_purchase_orders WHERE po_number = $1)`, poNumber).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("checking purchase order existence: %w", err)
	}
	if !exists {
		return nil, ErrPurchaseOrderNotFound
	}

	rows, err := pool.Query(ctx, `
		SELECT c.comment_id, c.po_number, c.author_id, u.name, c.comment_text, c.created_at
		FROM finance_order_comments c
		JOIN finance_users u ON u.user_id = c.author_id
		WHERE c.po_number = $1
		ORDER BY c.created_at ASC
	`, poNumber)
	if err != nil {
		return nil, fmt.Errorf("querying comments: %w", err)
	}
	defer rows.Close()

	comments := []Comment{}
	for rows.Next() {
		var cm Comment
		if err := rows.Scan(&cm.CommentID, &cm.PONumber, &cm.AuthorID, &cm.AuthorName, &cm.CommentText, &cm.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning comment: %w", err)
		}
		comments = append(comments, cm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating comments: %w", err)
	}

	return comments, nil
}

func AddComment(ctx context.Context, pool *pgxpool.Pool, poNumber, authorID, commentText string) (*Comment, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM finance_purchase_orders WHERE po_number = $1)`, poNumber).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("checking purchase order existence: %w", err)
	}
	if !exists {
		return nil, ErrPurchaseOrderNotFound
	}

	var commentID int64
	var createdAt time.Time
	err = pool.QueryRow(ctx, `
		INSERT INTO finance_order_comments (po_number, author_id, comment_text)
		VALUES ($1, $2, $3)
		RETURNING comment_id, created_at
	`, poNumber, authorID, commentText).Scan(&commentID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("inserting comment: %w", err)
	}

	var authorName string
	err = pool.QueryRow(ctx, `SELECT name FROM finance_users WHERE user_id = $1`, authorID).Scan(&authorName)
	if err != nil {
		return nil, fmt.Errorf("fetching author name: %w", err)
	}

	return &Comment{
		CommentID:   commentID,
		PONumber:    poNumber,
		AuthorID:    authorID,
		AuthorName:  authorName,
		CommentText: commentText,
		CreatedAt:   createdAt,
	}, nil
}

func DeleteComment(ctx context.Context, pool *pgxpool.Pool, commentID int64) error {
	result, err := pool.Exec(ctx, `DELETE FROM finance_order_comments WHERE comment_id = $1`, commentID)
	if err != nil {
		return fmt.Errorf("deleting comment: %w", err)   
	}
	if result.RowsAffected() == 0 {
		return ErrCommentNotFound
	}
	return nil
}
