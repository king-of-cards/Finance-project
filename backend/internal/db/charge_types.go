package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChargeType struct {
	ChargeTypeID   string  `json:"charge_type_id"`
	Name           string  `json:"name"`
	DefaultRateMin float64 `json:"default_rate_min"`
	DefaultRateMax float64 `json:"default_rate_max"`
}

var ErrChargeTypeNotFound = errors.New("charge type not found")
var ErrChargeTypeInUse = errors.New("charge type is in use and cannot be deleted")

func GetChargeTypes(ctx context.Context, pool *pgxpool.Pool) ([]ChargeType, error) {
	rows, err := pool.Query(ctx, `SELECT charge_type_id, name, default_rate_min, default_rate_max FROM finance_charge_types ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("querying charge types: %w", err)
	}
	defer rows.Close()

	var types []ChargeType
	for rows.Next() {
		var ct ChargeType
		if err := rows.Scan(&ct.ChargeTypeID, &ct.Name, &ct.DefaultRateMin, &ct.DefaultRateMax); err != nil {
			return nil, fmt.Errorf("scanning charge type: %w", err)
		}
		types = append(types, ct)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating charge types: %w", err)
	}

	return types, nil
}

func CreateChargeType(ctx context.Context, pool *pgxpool.Pool, chargeTypeID, name string, min, max float64) (*ChargeType, error) {
	var ct ChargeType
	err := pool.QueryRow(ctx, `
		INSERT INTO finance_charge_types (charge_type_id, name, default_rate_min, default_rate_max)
		VALUES ($1, $2, $3, $4)
		RETURNING charge_type_id, name, default_rate_min, default_rate_max
	`, chargeTypeID, name, min, max).Scan(&ct.ChargeTypeID, &ct.Name, &ct.DefaultRateMin, &ct.DefaultRateMax)
	if err != nil {
		return nil, fmt.Errorf("creating charge type: %w", err)
	}
	return &ct, nil
}

type UpdateChargeTypeInput struct {
	Name           *string
	DefaultRateMin *float64
	DefaultRateMax *float64
}

func UpdateChargeType(ctx context.Context, pool *pgxpool.Pool, chargeTypeID string, input UpdateChargeTypeInput) (*ChargeType, error) {
	var setClauses []string
	var args []interface{}
	argPos := 1
	addField := func(column string, value interface{}) {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, argPos))
		args = append(args, value)
		argPos++
	}
	if input.Name != nil {
		addField("name", *input.Name)
	}
	if input.DefaultRateMin != nil {
		addField("default_rate_min", *input.DefaultRateMin)
	}
	if input.DefaultRateMax != nil {
		addField("default_rate_max", *input.DefaultRateMax)
	}

	if len(setClauses) == 0 {
		return nil, ErrNoFieldsToUpdate
	}

	query := fmt.Sprintf(`
		UPDATE finance_charge_types SET %s WHERE charge_type_id = $%d
		RETURNING charge_type_id, name, default_rate_min, default_rate_max
	`, joinComma(setClauses), argPos)
	args = append(args, chargeTypeID)

	var ct ChargeType
	err := pool.QueryRow(ctx, query, args...).Scan(&ct.ChargeTypeID, &ct.Name, &ct.DefaultRateMin, &ct.DefaultRateMax)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrChargeTypeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("updating charge type: %w", err)
	}

	return &ct, nil
}

func DeleteChargeType(ctx context.Context, pool *pgxpool.Pool, chargeTypeID string) error {
	var inUse bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM finance_line_item_charges WHERE charge_type_id = $1)`, chargeTypeID).Scan(&inUse)
	if err != nil {
		return fmt.Errorf("checking charge type usage: %w", err)
	}
	if inUse {
		return ErrChargeTypeInUse
	}

	result, err := pool.Exec(ctx, `DELETE FROM finance_charge_types WHERE charge_type_id = $1`, chargeTypeID)
	if err != nil {
		return fmt.Errorf("deleting charge type: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrChargeTypeNotFound
	}

	return nil
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}
