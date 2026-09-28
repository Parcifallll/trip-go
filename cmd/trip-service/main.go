package main

import (
	"context"
	"log"

	"github.com/Parcifallll/trip-go/internal/config"
	"github.com/Parcifallll/trip-go/internal/handler"
	"github.com/Parcifallll/trip-go/internal/repository"
	"github.com/Parcifallll/trip-go/internal/server"
	"github.com/Parcifallll/trip-go/internal/txmanager"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to parse database URL: %v", err)
	}
	poolConfig.MaxConns = cfg.DatabaseMaxConns
	poolConfig.MinConns = cfg.DatabaseMinConns
	poolConfig.MaxConnLifetime = cfg.DatabaseMaxConnLifetime

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DatabaseConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("failed to create connection pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	log.Printf("database connection established")

	tripRepo := repository.NewTripRepository(pool)
	statusHistoryRepo := repository.NewStatusHistoryRepository(pool)

	txManager := txmanager.NewTxManager(pool)

	tripHandler := handler.NewTripHandler(txManager, tripRepo, statusHistoryRepo)
	healthHandler := handler.NewHealthHandler(pool)

	srv := server.NewServer(server.Config{
		HTTPAddr:        cfg.HTTPAddr,
		ShutdownTimeout: cfg.ShutdownTimeout,
	}, tripHandler, healthHandler)

	log.Printf("starting server on %s", cfg.HTTPAddr)
	if err := srv.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}

	log.Printf("server stopped gracefully")
}
