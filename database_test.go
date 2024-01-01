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

			_, err = connectDB(ctx)

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

func TestGetDBName(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
		want  string
	}{
		{
			name:  "Successful Get DB Name by Default",
			setup: func() {},
			want:  "mydatabase",
		},
		{
			name: "Successful Get DB Name by Env",
			setup: func() {
				// set env
				t.Setenv("MYSQL_DATABASE", "mydatabase2")
			},
			want: "mydatabase2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got := getDBName()
			if got != tt.want {
				t.Errorf("getDBName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
		want  string
	}{
		{
			name:  "Successful Get User by Default",
			setup: func() {},
			want:  "user",
		},
		{
			name: "Successful Get User by Env",
			setup: func() {
				// set env
				t.Setenv("MYSQL_USER", "user2")
			},
			want: "user2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got := getUser()
			if got != tt.want {
				t.Errorf("getUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPassword(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
		want  string
	}{
		{
			name:  "Successful Get Password by Default",
			setup: func() {},
			want:  "password",
		},
		{
			name: "Successful Get Password by Env",
			setup: func() {
				// set env
				t.Setenv("MYSQL_PASSWORD", "password2")
			},
			want: "password2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got := getPassword()
			if got != tt.want {
				t.Errorf("getPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAddr(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
		want  string
	}{
		{
			name:  "Successful Get Addr by Default",
			setup: func() {},
			want:  "localhost:3306",
		},
		{
			name: "Successful Get Addr by Env",
			setup: func() {
				// set env
				t.Setenv("MYSQL_HOST", "localhost2")
				t.Setenv("MYSQL_PORT", "33062")
			},
			want: "localhost2:33062",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got := getAddr()
			if got != tt.want {
				t.Errorf("getAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}
