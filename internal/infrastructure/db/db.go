package db

import (
	"flows/internal/domain"
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDB() (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	driver := os.Getenv("DB_DRIVER")
	dsn := os.Getenv("DB_DSN")

	if driver == "postgres" {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	} else {
		// Fallback por defecto a SQLite si no hay driver especificado
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" {
			dbPath = "flows.db"
		}
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// Auto Migrate the schema
	err = db.AutoMigrate(
		&domain.Connector{},
		&domain.ConnectorConfig{},
		&domain.Flow{},
		&domain.Execution{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}
