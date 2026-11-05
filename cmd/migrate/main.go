package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/voidxo/voidxo/internal/platform/config"
	"github.com/voidxo/voidxo/internal/platform/database"
	"github.com/voidxo/voidxo/migrations"
)

func main() {
	direction := flag.String("direction", "up", "migration direction: up or down")
	forceVersion := flag.Int("force", -2, "force migration version; use -1 to clear a dirty local state")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	dsn, err := migrationDSN(cfg.DatabaseDSN)
	if err != nil {
		logger.Error("invalid database DSN", "error", err)
		os.Exit(1)
	}
	db, err := database.OpenMySQL(context.Background(), dsn)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		logger.Error("load migration source", "error", err)
		os.Exit(1)
	}
	driver, err := migratemysql.WithInstance(db, &migratemysql.Config{MigrationsTable: database.TablePrefix + "schema_migrations"})
	if err != nil {
		logger.Error("create migration driver", "error", err)
		os.Exit(1)
	}
	runner, err := migrate.NewWithInstance("iofs", source, "mysql", driver)
	if err != nil {
		logger.Error("create migration runner", "error", err)
		os.Exit(1)
	}

	if *forceVersion >= -1 {
		if err := runner.Force(*forceVersion); err != nil {
			logger.Error("force migration version failed", "error", err)
			os.Exit(1)
		}
		logger.Info("migration version forced", "version", *forceVersion)
		return
	}

	switch *direction {
	case "up":
		err = runner.Up()
	case "down":
		err = runner.Down()
	default:
		logger.Error("invalid migration direction", "direction", *direction)
		os.Exit(1)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	logger.Info("database migration completed", "direction", *direction)
}

func migrationDSN(dsn string) (string, error) {
	parsed, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	// Migration files contain multiple statements; the API DSN intentionally remains single-statement.
	parsed.MultiStatements = true
	return parsed.FormatDSN(), nil
}
