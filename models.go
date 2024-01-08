package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

type DataItem interface {
	BulkInsert(ctx context.Context, db *sql.DB) error
}

type DataItems []DataItem

type PanelOrderItem struct {
	PanelOrderID int `fake:"{number:1,100}"`
	QuestionID   int `fake:"{number:1,100}"`
	OrderIndex   int `fake:"{number:1,100}"`
}

type PanelOrderItems []PanelOrderItem

func (ps PanelOrderItems) BulkInsert(ctx context.Context, db *sql.DB) error {
	// Create query for bulk insert
	var query strings.Builder
	query.WriteString("INSERT INTO panel_order_items (panel_order_id, question_id, order_index) VALUES ")
	for i := 0; i < len(ps); i++ {
		query.WriteString("(?, ?, ?)")
		if i != len(ps)-1 {
			query.WriteString(", ")
		}
	}

	// Create params for bulk insert query
	var params []interface{}
	for _, p := range ps {
		params = append(params, p.PanelOrderID, p.QuestionID, p.OrderIndex)
	}

	txn, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		// rollback if panic
		if p := recover(); p != nil {
			if err := txn.Rollback(); err != nil && err != sql.ErrTxDone {
				slog.ErrorContext(ctx, "failed to rollback transaction after panic", "error", err)
			}
			panic(p) // re-throw panic after Rollback
		} else if rerr := txn.Rollback(); rerr != nil && rerr != sql.ErrTxDone {
			// err is non-nil; stop the panic and return error
			slog.ErrorContext(ctx, "failed to rollback transaction", "error", rerr)
			return
		}
	}()

	result, err := txn.ExecContext(ctx, query.String(), params...)
	if err != nil {
		return fmt.Errorf("failed to insert multiple records: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	slog.DebugContext(ctx, "inserted multiple records", "rows_affected", rows, "last_insert_id", lastID)

	return txn.Commit()
}
