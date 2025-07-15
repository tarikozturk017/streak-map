package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"github.com/tarikozturk017/streak-map/backend/internal/models"
)

type DB struct {
	*gorm.DB
}

func NewConnection(host, port, user, password, dbname string) (*DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user, password, dbname, port)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) AutoMigrate() error {
	err := db.DB.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}
	
	log.Println("Database migration completed successfully")
	return nil
}