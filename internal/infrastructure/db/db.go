package db

import (
	"flows/internal/domain"
	"fmt"
	"os"
	"time"

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
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			PrepareStmt: true, // Mejora rendimiento al cachear sentencias SQL
		})
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

	// Configuración del Pool de Conexiones
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get generic database object: %w", err)
	}

	// SetMaxIdleConns establece el número máximo de conexiones en el pool de conexiones inactivas.
	sqlDB.SetMaxIdleConns(20)

	// SetMaxOpenConns establece el número máximo de conexiones abiertas a la base de datos.
	sqlDB.SetMaxOpenConns(90)

	// SetConnMaxLifetime establece la cantidad máxima de tiempo que una conexión puede ser reutilizada.
	sqlDB.SetConnMaxLifetime(time.Minute * 5)

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
