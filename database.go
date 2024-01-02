package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

type DBConfig struct {
	DBName   string
	User     string
	Password string
	Addr     string
}

type sqlOpenFunc func(driverName, dataSourceName string) (*sql.DB, error)

var sqlOpen sqlOpenFunc = sql.Open

func NewDBConfig() *DBConfig {
	return &DBConfig{
		DBName:   getEnv("MYSQL_DATABASE", "mydatabase"),
		User:     getEnv("MYSQL_USER", "user"),
		Password: getEnv("MYSQL_PASSWORD", "password"),
		Addr:     getEnvAddr("MYSQL_HOST", "MYSQL_PORT", "localhost", "3306"),
	}
}

func connectDB(ctx context.Context, config *DBConfig) (*sql.DB, error) {
	c := mysql.Config{
		DBName:    config.DBName,
		User:      config.User,
		Passwd:    config.Password,
		Addr:      config.Addr,
		Net:       "tcp",
		ParseTime: true,
		Collation: "utf8mb4_general_ci",
		Loc:       time.Local,
	}

	db, err := sqlOpen("mysql", c.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("Failed to open the database: %v", err)
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("Failed to ping the database: %v", err)
	}
	return db, nil
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func getEnvAddr(hostKey, portKey, hostDefault, portDefault string) string {
	host := getEnv(hostKey, hostDefault)
	port := getEnv(portKey, portDefault)
	return strings.Join([]string{host, port}, ":")
}
