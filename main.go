package main

import (
	"context"
	"database/sql"
	"flag"
	"time"

	"github.com/bobcob7/polly-bot/internal"
	"go.uber.org/zap"
)

var configFilename string
var logger *zap.Logger

func init() {
	flag.StringVar(&configFilename, "config", "", "File to the configuration file")
	logger, _ = zap.NewProduction()
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
	err = internal.InitDB(db, conf.InitDemo)
	if err != nil {
		logger.Fatal("Failed to initialize database",
			zap.Bool("initDemo", conf.InitDemo),
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
