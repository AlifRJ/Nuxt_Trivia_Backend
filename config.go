package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func initPostgres(ctx context.Context) *pgxpool.Pool {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")),
		Host:   fmt.Sprintf("%s:%s", os.Getenv("DB_HOST"), os.Getenv("DB_PORT")),
		Path:   os.Getenv("DB_DATABASE"),
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	// Alternative URL parameter setting:
	// q.Set("default_query_exec_mode", "simple_protocol")
	u.RawQuery = q.Encode()

	config, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		log.Fatalf("Error parsing database DSN: %v\n", err)
	}

	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 15 * time.Minute

	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Error creating database pool: %v\n", err)
	}

	// Verify physical connection to database
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.Ping(pingCtx); err != nil {
		log.Fatalf("Cannot connect to database: %v\n", err)
	}

	fmt.Println("Successfully connected to PostgreSQL!")
	return db
}

func initRedis(ctx context.Context) *redis.Client {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisDatabase := os.Getenv("REDIS_DATABASE")

	db, err := strconv.Atoi(redisDatabase)
	if err != nil {
		db = 0
	}

	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Username: os.Getenv("REDIS_USER"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	}

	// Enable TLS only if REDIS_USE_TLS is set to "true"
	if useTLS, _ := strconv.ParseBool(os.Getenv("REDIS_USE_TLS")); useTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: redisHost,
		}
	}

	rdb := redis.NewClient(opts)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		log.Fatalf("Cannot connect to Redis: %v\n", err)
	}

	fmt.Println("Successfully connected to Redis!")
	return rdb
}
