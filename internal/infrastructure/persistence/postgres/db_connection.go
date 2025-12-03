package postgres

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBConfig holds the database configuration.
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DSN returns the Data Source Name string.
func (c *DBConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

var (
	dbPool *pgxpool.Pool
	once   sync.Once
)

// GetDBPool returns the database connection pool instance.
func GetDBPool(ctx context.Context, config *DBConfig) (*pgxpool.Pool, error) {
	var err error
	once.Do(func() {
		var poolConfig *pgxpool.Config
		poolConfig, err = pgxpool.ParseConfig(config.DSN())
		if err != nil {
			return
		}

		// Connection pool settings
		poolConfig.MaxConns = 25
		poolConfig.MinConns = 5
		poolConfig.MaxConnLifetime = 5 * time.Minute
		poolConfig.MaxConnIdleTime = 1 * time.Minute

		dbPool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	if err := dbPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return dbPool, nil
}

// CloseDBPool closes the database connection pool.
func CloseDBPool() {
	if dbPool != nil {
		dbPool.Close()
	}
}
