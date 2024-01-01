package main

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

type sqlOpenFunc func(driverName, dataSourceName string) (*sql.DB, error)

var sqlOpen sqlOpenFunc = sql.Open

func connectDB(ctx context.Context) (*sql.DB, error) {
	c := mysql.Config{
		DBName:    getDBName(),
		User:      getUser(),
		Passwd:    getPassword(),
		Addr:      getAddr(),
		Net:       "tcp",
		ParseTime: true,
		Collation: "utf8mb4_general_ci",
		Loc:       time.Local,
	}

	db, err := sqlOpen("mysql", c.FormatDSN())
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

func getDBName() string {
	dbname := os.Getenv("MYSQL_DATABASE")
	if dbname == "" {
		dbname = "mydatabase"
	}
	return dbname
}

func getUser() string {
	user := os.Getenv("MYSQL_USER")
	if user == "" {
		user = "user"
	}
	return user
}

func getPassword() string {
	password := os.Getenv("MYSQL_PASSWORD")
	if password == "" {
		password = "password"
	}
	return password
}

func getAddr() string {
	host := os.Getenv("MYSQL_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("MYSQL_PORT")
	if port == "" {
		port = "3306"
	}
	return strings.Join([]string{host, port}, ":")
}
