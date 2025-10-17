package database

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DefaultConfig returns default database configuration
func DefaultConfig() *Config {
	config := &Config{
		Host:     "localhost",
		Port:     5433, // Updated to match Docker setup
		User:     "openusp",
		Password: "openusp123",
		DBName:   "openusp_db",
		SSLMode:  "disable",
	}

	// Override with environment variables if present
	if host := os.Getenv("DB_HOST"); host != "" {
		config.Host = host
	}
	if portStr := os.Getenv("DB_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.Port = port
		}
	}
	if user := os.Getenv("DB_USER"); user != "" {
		config.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		config.Password = password
	}
	if dbname := os.Getenv("DB_NAME"); dbname != "" {
		config.DBName = dbname
	}
	if sslmode := os.Getenv("DB_SSLMODE"); sslmode != "" {
		config.SSLMode = sslmode
	}

	return config
}

// Database represents the database connection and operations
type Database struct {
	DB     *gorm.DB
	config *Config
}

// NewDatabase creates a new database instance
func NewDatabase(config *Config) (*Database, error) {
	if config == nil {
		config = DefaultConfig()
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &Database{
		DB:     db,
		config: config,
	}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping tests the database connection
func (d *Database) Ping() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// Migrate runs database migrations
func (d *Database) Migrate() error {
	log.Println("Running database migrations...")

	// Migrate all models
	err := d.DB.AutoMigrate(
		&Device{},
		&Parameter{},
		&Alert{},
		&Session{},
		&ConnectionHistory{},
	)

	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database migration completed successfully")
	return nil
}

// GetStats returns database statistics
func (d *Database) GetStats() map[string]interface{} {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
	}
}
