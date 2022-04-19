package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bobcob7/polly/internal/config"
	"github.com/bobcob7/polly/internal/echo"
	"github.com/bobcob7/polly/internal/mapper"
	"github.com/bobcob7/polly/internal/ping"
	"github.com/bobcob7/polly/internal/whoami"
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
	flag.BoolVar(&showVersion, "version", false, "Print current version and exit")
}

func main() {
	flag.Parse()
	if showVersion {
		fmt.Println("0.0.1")
		os.Exit(0)
	}
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
	// Start transmission interface
	tr, err := transmission.New(ctx, tranmissionEndpoint)
	if err != nil {
		zap.L().Fatal("failed to connect to tranmission RPC server", zap.Error(err))
	}
	// Start discord interface
	bot := discord.New(
		&whoami.WhoAmI{},
		&echo.Echo{},
		&ping.Ping{},
		&transmission.AddDownload{Transmission: tr},
		&transmission.UnfinishedDownloads{Transmission: tr},
		&transmission.SubscribeDownloads{Transmission: tr},
	)
	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)
		errChan <- bot.Run(ctx, cfg.Discord.Token, cfg.Discord.GuildID)
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
