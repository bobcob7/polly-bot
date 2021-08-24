package main

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"time"

	"github.com/bobcob7/polly-bot/internal"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"
)

var configFilename string
var logger *zap.Logger

func init() {
	flag.StringVar(&configFilename, "config", "", "File to the configuration file")
	logger, _ = zap.NewProduction()
}

//go:embed migrations/*.sql
var migrations embed.FS

type Logger struct {
	*zap.Logger
}

func (l Logger) Printf(format string, values ...interface{}) {
	l.Info(fmt.Sprintf(format, values...))
}

func (l Logger) Verbose() bool {
	return true
}

func InitDB(dbURL string) error {
	logger.Info("Initializing database connection")

	d, err := iofs.New(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migrations: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	m.Log = Logger{logger}
	err = m.Up()
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

func main() {
	defer logger.Sync()
	flag.Parse()
	ctx, done := context.WithCancel(context.Background())
	defer done()

	// Open config
	logger.Info("Opening config file",
		zap.String("filename", configFilename),
	)
	conf, err := internal.OpenConfig(configFilename)
	if err != nil {
		logger.Fatal("Failed to open config file",
			zap.Error(err),
		)
	}
	// Connect to DB
	db, err := sql.Open("postgres", conf.ConnectionString)
	if err != nil {
		logger.Fatal("Failed to open database",
			zap.Error(err),
		)
	}
	logger.Info("Opened database connection")
	err = InitDB(conf.ConnectionString)
	if err != nil {
		logger.Fatal("Failed to initialize database",
			zap.Error(err),
		)
	}
	// Create discord bot
	logger.Info("Creating discord bot")
	discord, err := internal.NewDiscordController(conf.DiscordToken)
	if err != nil {
		logger.Fatal("Failed to initialize discord bot",
			zap.Error(err),
		)
	}
	defer discord.Close()
	go discord.Run(ctx, db)
	// Create scanner components
	history := internal.NewMemoryHistory(conf.HistoryLength)
	downloader := internal.NewSingleDownloader(conf.DownloadDirectory)
	// Assemble scanner
	scanner := internal.NewScanner(history, downloader)

	// Process subjects in loop
	period, _ := time.ParseDuration(conf.RSSPeriod)

	logger.Info("Running scanner")
	scanner.Run(ctx, db, period)
}
