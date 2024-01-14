package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/brianvoe/gofakeit/v6"
)

type PanelOrderItem struct {
	PanelOrderID int64 `fake:"{number:1,1462491394}"`
	QuestionID   int   `fake:"{number:1,3117100}"`
	OrderIndex   int   `fake:"{number:0,1000}"`
}

type PanelOrderItems []PanelOrderItem

type DataItemCreator interface {
	Create() (DataItems, error) // Create returns a slice of DataItem and error if the creation of DataItem fails due to any reason
}

type PanelOrderItemCreator struct{}

func (p *PanelOrderItemCreator) Create() (DataItems, error) {
	var item PanelOrderItem
	if err := gofakeit.Struct(&item); err != nil {
		return nil, fmt.Errorf("failed to create PanelOrderItem: %w", err)
	}
	return DataItems{PanelOrderItems{item}}, nil
}

type DataItem interface {
	BulkInsert(ctx context.Context, db *sql.DB) error // BulkInsert inserts the data item into the database and error if the bulk insertion of DataItems fails due to any reason
}

type DataItems []DataItem

func (ps PanelOrderItems) BulkInsert(ctx context.Context, db *sql.DB) error {
	query, params := buildInsertQuery(ps)

	txn, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		handleTransaction(ctx, txn)
	}()

	if err := executeInsert(ctx, txn, query, params); err != nil {
		return err
	}

	return txn.Commit()
}

// buildInsertQuery builds the SQL query and parameters for bulk insert.
func buildInsertQuery(ps PanelOrderItems) (string, []interface{}) {
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

	return query.String(), params
}

// executeInsert executes the insert query in the given transaction.
func executeInsert(ctx context.Context, txn *sql.Tx, query string, params []interface{}) error {
	result, err := txn.ExecContext(ctx, query, params...)
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

	return nil
}

// handleTransaction handles the finalization of the transaction.
func handleTransaction(ctx context.Context, txn *sql.Tx) {
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
}
