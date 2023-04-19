package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bobcob7/polly-bot/internal/commands"
	"github.com/bobcob7/polly-bot/internal/config"
	"github.com/bobcob7/polly-bot/internal/mapper"
	"github.com/bobcob7/polly-bot/internal/server"
	"github.com/bobcob7/polly-bot/pkg/discord"
	"github.com/bobcob7/transmission-rpc"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	"github.com/upper/db/v4"
	"go.uber.org/zap"
)

var showVersion bool

//go:embed migrations/*.sql
var migrationFiles embed.FS

func getDatabase(cfg config.Database) (db.Session, error) {
	// Connect to DB
	sess, err := cfg.Session()
	if err != nil {
		return nil, fmt.Errorf("failed creating db session: %w", err)
	}
	// Perform migration
	// driver, err := postgres.WithInstance(sess, &postgres.Config{})
	// m, err := migrate.NewWithDatabaseInstance(
	// 	"file:///migrations",
	// 	"postgres", driver)
	// m.Up() // or m.Step(2) if you want to explicitly set the number of migrations to run
	driver, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed creating migration driver: %w", err)
	}
	connString, err := cfg.URL()
	if err != nil {
		return nil, fmt.Errorf("failed getting configured db url: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", driver, connString)
	if err != nil {
		return nil, fmt.Errorf("failed creating new migration source: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("failed to run migration: %w", err)
	}
	return sess, nil
}

func init() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
	flag.BoolVar(&showVersion, "version", false, "Print current version and exit")
}

func main() {
	flag.Parse()
	cfg := config.New()
	dec := mapper.NewDecoder(os.LookupEnv, mapper.WithTagDefaulter(strings.ToUpper))
	if err := dec.Decode(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	errs := cfg.Valid()
	if !errs.Ok() {
		fmt.Fprintln(os.Stderr, errs.Error())
		os.Exit(1)
	}
	if cfg.Discord.GuildID == "" {
		guilds, err := discord.GetGuilds(cfg.Discord.Token)
		if err != nil {
			zap.L().Fatal("failed to get guilds", zap.Error(err))
		}
		for id, name := range guilds {
			zap.L().Info("guild", zap.String("id", id), zap.String("name", name))
		}
		return
	}
	ctx, done := context.WithCancel(context.Background())
	defer done()
	// Start database connection
	pool, err := getDatabase(cfg.Database)
	if err != nil {
		zap.L().Fatal("failed to connect to database", zap.Error(err))
	}
	// Start transmission interface
	transmissionClient, err := transmission.New(ctx, cfg.Transmission.Endpoint)
	if err != nil {
		zap.L().Fatal("failed to connect to transmission RPC server", zap.Error(err))
	}
	// Start transmission/db interface
	srv := server.New(cfg, pool, transmissionClient)
	go func() {
		if err := srv.Run(ctx); err != nil {
			zap.L().Fatal("failed to startup RPC server", zap.Error(err))
		}
	}()

	getAll := commands.NewGetAllCommand(pool)
	addTorrent := commands.NewAddCommand(pool, transmissionClient)

	// Start discord interface
	bot := discord.New(
		cfg.Discord,
		&commands.WhoAmI{},
		&commands.Echo{},
		&commands.Ping{},
		getAll,
		addTorrent,

		// &transmission.AddDownload{Transmission: tr},
		// &transmission.UnfinishedDownloads{Transmission: tr},
		// &transmission.SubscribeDownloads{Transmission: tr},
	)
	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)
		errChan <- bot.Run(ctx)
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	for {
		select {
		case err, ok := <-errChan:
			if !ok {
				// Exit since the error channel is closed
				return
			}
			if err != nil {
				zap.L().Error("Error running bot", zap.Error(err))
			}
		case <-stop:
			zap.L().Info("Received interrupt, exiting")
			done()
		}
	}
}
