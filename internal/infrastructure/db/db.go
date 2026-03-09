package db

import (
	"flows/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDB() (*gorm.DB, error) {
	// Using SQLite for standalone demonstration of the Flows module
	// In production, this should be MySQL or PostgreSQL
	db, err := gorm.Open(sqlite.Open("flows.db"), &gorm.Config{})
	if err != nil {
		return nil, err
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
