package main

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPanelOrderItemsBulkInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	tests := []struct {
		name    string
		items   PanelOrderItems
		wantErr bool
	}{
		{
			name: "Successful Insert",
			items: PanelOrderItems{
				{PanelOrderID: 1, QuestionID: 1, OrderIndex: 1},
				{PanelOrderID: 2, QuestionID: 2, OrderIndex: 2},
			},
			wantErr: false,
		},
		{
			name: "Failed Insert",
			items: PanelOrderItems{
				{PanelOrderID: 1, QuestionID: 1, OrderIndex: 1},
				{PanelOrderID: 1, QuestionID: 1, OrderIndex: 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var mockArgs []driver.Value
			for _, item := range tt.items {
				mockArgs = append(mockArgs, item.PanelOrderID, item.QuestionID, item.OrderIndex)
			}

			mock.ExpectBegin()

			if tt.wantErr {
				mock.ExpectExec("INSERT INTO panel_order_items").WithArgs(mockArgs...).WillReturnError(fmt.Errorf("failed to insert multiple records"))
				mock.ExpectRollback()
			} else {
				mock.ExpectExec("INSERT INTO panel_order_items").WithArgs(mockArgs...).WillReturnResult(sqlmock.NewResult(1, int64(len(tt.items))))
				mock.ExpectCommit()
			}

			// BulkInsert関数を呼び出し
			err := tt.items.BulkInsert(ctx, db)
			if (err != nil) != tt.wantErr {
				t.Errorf("PanelOrderItems.BulkInsert() error = %v, wantErr %v", err, tt.wantErr)
			}

			// モックの状態を検証
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
