package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"strings"
)

type PanelOrderItem struct {
	PanelOrderID int `fake:"{number:1,100}"`
	QuestionID   int `fake:"{number:1,100}"`
	OrderIndex   int `fake:"{number:1,100}"`
}

type PanelOrderItems []PanelOrderItem

func (ps PanelOrderItems) BulkInsert(ctx context.Context, db *sql.DB) (int64, error) {
	// バルクインサート用のクエリを作成
	var query strings.Builder
	query.WriteString("INSERT INTO panel_order_items (panel_order_id, question_id, order_index) VALUES ")
	for i := 0; i < len(ps); i++ {
		query.WriteString("(?, ?, ?)")
		if i != len(ps)-1 {
			query.WriteString(", ")
		}
	}

	// バルクインサート用のパラメータを作成
	var params []interface{}
	for _, p := range ps {
		params = append(params, p.PanelOrderID, p.QuestionID, p.OrderIndex)
	}

	txn, err := db.BeginTx(ctx, nil)
	if err != nil {
		return math.MinInt, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			if rerr := txn.Rollback(); rerr != nil {
				slog.ErrorContext(ctx, "failed to rollback transaction after panic", "error", rerr)
			}
			panic(p)
		} else if err != nil {
			if rerr := txn.Rollback(); rerr != nil {
				slog.ErrorContext(ctx, "failed to rollback transaction", "error", rerr)
			}
		}
	}()

	result, err := txn.ExecContext(ctx, query.String(), params...)
	if err != nil {
		return math.MinInt, fmt.Errorf("failed to insert multiple records: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return math.MinInt, fmt.Errorf("failed to get affected rows: %w", err)
	}

	if err := txn.Commit(); err != nil {
		return math.MinInt, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return rows, nil
}
