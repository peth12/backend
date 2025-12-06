package handlers

import (
	"log"
	"spendwise-backend/internal/database"
	"spendwise-backend/internal/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB() {
	// Connection string for the default 'postgres' database to create the test db
	dsn := "host=localhost user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=Asia/Bangkok"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("Failed to connect to postgres database:", err)
	}

	// Create test database if not exists
	// Note: This might fail if connected to the db we are trying to drop/create.
	// We are connected to 'postgres', so dropping 'spendwise_test' is fine.
	// But we need to close connections to it first?
	// For simplicity, we just try to create it.

	// Check if db exists
	var count int64
	db.Raw("SELECT count(*) FROM pg_database WHERE datname = ?", "spendwise_test").Scan(&count)

	if count == 0 {
		// Create database
		// GORM doesn't support CREATE DATABASE directly via AutoMigrate, need Raw SQL
		// And usually cannot execute CREATE DATABASE inside a transaction block.
		// So we need to ensure we are not in a transaction? GORM open doesn't start one.
		if err := db.Exec("CREATE DATABASE spendwise_test").Error; err != nil {
			log.Fatal("Failed to create test database:", err)
		}
	}

	// Now connect to the test database
	testDSN := "host=localhost user=postgres password=password dbname=spendwise_test port=5432 sslmode=disable TimeZone=Asia/Bangkok"
	testDB, err := gorm.Open(postgres.Open(testDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("Failed to connect to test database:", err)
	}

	// Clean up database (Drop tables to ensure fresh state)
	// Or just Migrate? Better to drop to avoid stale data.
	// But dropping tables one by one is tedious.
	// Let's just AutoMigrate and then truncate tables?
	// Or DropAll?

	// For now, let's just AutoMigrate. If we want fresh state, we should probably drop tables.
	// Let's drop the specific tables we use.
	testDB.Migrator().DropTable(&models.User{}, &models.ExpenseGroup{}, &models.GroupMember{}, &models.UserRole{}, &models.ExpenseRequest{}, &models.ExpenseAttachment{}, &models.ApprovalSlip{})

	// Migrate schema
	err = testDB.AutoMigrate(
		&models.User{},
		&models.ExpenseGroup{},
		&models.GroupMember{},
		&models.UserRole{},
		&models.ExpenseRequest{},
		&models.ExpenseAttachment{},
		&models.ApprovalSlip{},
	)
	if err != nil {
		log.Fatal("Failed to migrate test database:", err)
	}

	database.DB = testDB
}

func setupApp() *fiber.App {
	app := fiber.New()
	return app
}
