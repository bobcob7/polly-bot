package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bobcob7/polly/pkg/discord"
	"github.com/bobcob7/polly/pkg/transmission"
	"go.uber.org/zap"
)

var tranmissionEndpoint string
var showVersion bool

func init() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
	flag.StringVar(&tranmissionEndpoint, "endpoint", "https://transmission.bobcob7.com", "URL where transmission RPC can be reached")
	flag.BoolVar(&showVersion, "version", false, "Print current version and exit")
}

func main() {
	flag.Parse()
	if showVersion {
		fmt.Println("0.0.1")
		os.Exit(0)
	}
	// Authentication Token pulled from environment variable DGU_TOKEN
	token := os.Getenv("TOKEN")
	guildID := os.Getenv("GUILD")
	if guildID == "" {
		guilds, err := discord.GetGuilds(token)
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
	// Start transmission interface
	tr, err := transmission.New(ctx, tranmissionEndpoint)
	if err != nil {
		zap.L().Fatal("failed to connect to tranmission RPC server", zap.Error(err))
	}
	// Start discord interface
	bot := discord.New(
		&transmission.AddDownload{Transmission: tr},
		&transmission.UnfinishedDownloads{Transmission: tr},
		&transmission.SubscribeDownloads{Transmission: tr},
	)
	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)
		errChan <- bot.Run(ctx, token, guildID)
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
