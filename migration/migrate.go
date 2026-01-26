package migration

import (
	"database/sql"
	"errors"
	"os"

	"github.com/angpaoprw/cryptocurrency/logger"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

// RunMigrations runs database migrations
func RunMigrations(db *sql.DB) error {
	logger.Info("Starting database migrations...")

	// Create postgres driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logger.Error("Failed to create postgres driver for migrations",
			zap.Error(err),
		)
		return err
	}

	// Get migration path
	migrationPath := "file://db/migration"
	if _, err := os.Stat("db/migration"); os.IsNotExist(err) {
		logger.Error("Migration directory not found",
			zap.String("path", "db/migration"),
			zap.Error(err),
		)
		return err
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(migrationPath, "postgres", driver)
	if err != nil {
		logger.Error("Failed to create migrate instance",
			zap.String("migration_path", migrationPath),
			zap.Error(err),
		)
		return err
	}

	// Get current version
	currentVersion, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		logger.Error("Failed to get current migration version",
			zap.Error(err),
		)
		return err
	}

	if dirty {
		logger.Warn("Database is in dirty state",
			zap.Uint("version", currentVersion),
		)
	}

	// Run migrations
	logger.Info("Running database migrations...",
		zap.Uint("current_version", currentVersion),
	)

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("No new migrations to run",
				zap.Uint("current_version", currentVersion),
			)
			return nil
		}
		logger.Error("Failed to run migrations",
			zap.Error(err),
		)
		return err
	}

	// Get new version
	newVersion, _, err := m.Version()
	if err != nil {
		logger.Error("Failed to get new migration version",
			zap.Error(err),
		)
		return err
	}

	logger.Info("Database migrations completed successfully",
		zap.Uint("previous_version", currentVersion),
		zap.Uint("new_version", newVersion),
	)

	return nil
}

// ForceMigrationVersion forces the migration version (use with caution)
func ForceMigrationVersion(db *sql.DB, version int) error {
	logger.Warn("Forcing migration version",
		zap.Int("version", version),
	)

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://db/migration", "postgres", driver)
	if err != nil {
		return err
	}

	err = m.Force(version)
	if err != nil {
		logger.Error("Failed to force migration version",
			zap.Int("version", version),
			zap.Error(err),
		)
		return err
	}

	logger.Info("Migration version forced successfully",
		zap.Int("version", version),
	)

	return nil
}

// DropDatabase drops all tables (use with extreme caution)
func DropDatabase(db *sql.DB) error {
	logger.Warn("Dropping all database tables - this will delete all data!")

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://db/migration", "postgres", driver)
	if err != nil {
		return err
	}

	err = m.Drop()
	if err != nil {
		logger.Error("Failed to drop database",
			zap.Error(err),
		)
		return err
	}

	logger.Info("Database dropped successfully")
	return nil
}

// GetMigrationStatus returns the current migration status
func GetMigrationStatus(db *sql.DB) (map[string]interface{}, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logger.Error("Failed to create postgres driver for migration status",
			zap.Error(err),
		)
		return nil, err
	}

	m, err := migrate.NewWithDatabaseInstance("file://db/migration", "postgres", driver)
	if err != nil {
		logger.Error("Failed to create migrate instance for status",
			zap.Error(err),
		)
		return nil, err
	}

	version, dirty, err := m.Version()
	status := map[string]interface{}{
		"version": nil,
		"dirty":   false,
		"error":   nil,
	}

	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			status["error"] = "No migrations have been run"
			logger.Info("No migrations have been run yet")
		} else {
			status["error"] = err.Error()
			logger.Error("Failed to get migration version",
				zap.Error(err),
			)
			return status, err
		}
	} else {
		status["version"] = version
		status["dirty"] = dirty

		if dirty {
			logger.Warn("Database is in dirty state",
				zap.Uint("version", version),
			)
		} else {
			logger.Info("Migration status retrieved",
				zap.Uint("version", version),
				zap.Bool("dirty", dirty),
			)
		}
	}

	return status, nil
}
