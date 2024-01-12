package main

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestConnectDB(t *testing.T) {
	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name:    "Successful Connection",
			setup:   func() {},
			wantErr: false,
		},
		{
			name: "Failed Connection",
			setup: func() {
				// set invalid db name
				t.Setenv("MYSQL_DATABASE", "invalidDB")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			// mock db connection
			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			if err != nil {
				t.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			// set mock
			if tt.wantErr {
				mock.ExpectPing().WillReturnError(sql.ErrConnDone)
			} else {
				mock.ExpectPing()
			}

			originalSQLOpen := sqlOpen
			defer func() { sqlOpen = originalSQLOpen }()
			sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
				return db, nil
			}

			ctx := context.Background()

			_, err = connectDB(ctx, NewDBConfig())
			if (err != nil) != tt.wantErr {
				t.Errorf("connectDB() error = %v, wantErr %v", err, tt.wantErr)
			}

			// verify mock
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
		want  string
	}{
		{
			name:  "Successful Get Env by Default",
			setup: func() {},
			want:  "default",
		},
		{
			name: "Successful Get Env by Env",
			setup: func() {
				// set env
				t.Setenv("TEST_ENV", "test")
			},
			want: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got := getEnv("TEST_ENV", "default")
			if got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvAddr(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
		want  string
	}{
		{
			name:  "Successful Get Env Addr by Default",
			setup: func() {},
			want:  "localhost:3306",
		},
		{
			name: "Successful Get Env Addr by Env",
			setup: func() {
				// set env
				t.Setenv("TEST_HOST", "test")
				t.Setenv("TEST_PORT", "1234")
			},
			want: "test:1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got := getEnvAddr("TEST_HOST", "TEST_PORT", "localhost", "3306")
			if got != tt.want {
				t.Errorf("getEnvAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvAsBool(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
		want  bool
	}{
		{
			name:  "Get Env Bool with false by Default",
			setup: func() {},
			want:  false,
		},
		{
			name: "Get Env Bool with true in lowercase",
			setup: func() {
				// set env
				t.Setenv("TEST_BOOL", "true")
			},
			want: true,
		},
		{
			name: "Get Env Bool with true in Titlecase",
			setup: func() {
				// set env
				t.Setenv("TEST_BOOL", "True")
			},
			want: true,
		},
		{
			name: "Get Env Bool with true in Uppercase",
			setup: func() {
				// set env
				t.Setenv("TEST_BOOL", "TRUE")
			},
			want: true,
		},
		{
			name: "Get Env Bool with false as 0",
			setup: func() {
				// set env
				t.Setenv("TEST_BOOL", "0")
			},
			want: false,
		},
		{
			name: "Get Env Bool with true as 1",
			setup: func() {
				// set env
				t.Setenv("TEST_BOOL", "1")
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got := getEnvAsBool("TEST_BOOL", false)
			if got != tt.want {
				t.Errorf("getEnvAsBool() = %v, want %v", got, tt.want)
			}
		})
	}
}
