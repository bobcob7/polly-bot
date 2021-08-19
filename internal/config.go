package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"go.uber.org/zap"
)

type Config struct {
	ConnectionString  string
	RSSPeriod         string
	HistoryLength     int
	DownloadDirectory string
	InitDemo          bool
	DiscordToken      string
}

type Errors struct {
	errs []error
}

func (e *Errors) Add(err error) {
	e.errs = append(e.errs, err)
}

func (e *Errors) Error() string {
	var output string
	for _, err := range e.errs {
		output += err.Error() + "\n"
	}
	return output
}

var schemaRe = regexp.MustCompile(`^([A-Za-z0-9]+):\/\/.*$`)

func (c Config) valid() (ok bool) {
	ok = true
	rssPeriod, err := time.ParseDuration(c.RSSPeriod)
	if err != nil {
		logger.Error("RSSPeriod must be a valid duration", zap.Error(err))
		ok = false
	} else if rssPeriod < time.Second*5 {
		logger.Error("RSSPeriod must be at least 5 seconds", zap.Duration("rssPeriod", rssPeriod))
		ok = false
	} else if rssPeriod > time.Hour*24 {
		logger.Error("RSSPeriod must be less than 24 hours", zap.Duration("rssPeriod", rssPeriod))
		ok = false
	}
	if c.HistoryLength < 10 {
		logger.Error("HistoryLength must be at least 10", zap.Int("HistoryLength", c.HistoryLength))
		ok = false
	} else if c.HistoryLength > 100000 {
		logger.Error("HistoryLength must be less than 100000", zap.Int("HistoryLength", c.HistoryLength))
		ok = false
	}
	if !schemaRe.MatchString(c.ConnectionString) {
		logger.Error("ConnectionString must have a valid schema", zap.String("ConnectionString", c.ConnectionString))
		ok = false
	}
	if c.DiscordToken == "" {
		logger.Error("DiscordToken is required")
		ok = false
	}
	return
}

func OpenConfig(filename string) (*Config, error) {
	var newConfig Config
	var err error
	if filename != "" {
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		decoder := json.NewDecoder(bufio.NewReader(f))
		err = decoder.Decode(&newConfig)
		if err != nil {
			return nil, err
		}
	}
	if v, ok := os.LookupEnv("CONNECTION_STRING"); ok {
		newConfig.ConnectionString = v
	}
	if v, ok := os.LookupEnv("RSS_PERIOD"); ok {
		newConfig.RSSPeriod = v
	}
	if v, ok := os.LookupEnv("HISTORY_LENGTH"); ok {
		newConfig.HistoryLength, err = strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("failed to convert HISTORY_LENGTH to int: %w", err)
		}
	}
	if v, ok := os.LookupEnv("DOWNLOAD_DIR"); ok {
		newConfig.DownloadDirectory = v
	}
	if v, ok := os.LookupEnv("INIT_DEMO"); ok {
		newConfig.InitDemo, err = strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("failed to convert INIT_DEMO to bool: %w", err)
		}
	}
	if v, ok := os.LookupEnv("DISCORD_TOKEN"); ok {
		newConfig.DiscordToken = v
	}
	if !newConfig.valid() {
		return nil, fmt.Errorf("invalid config")
	}
	return &newConfig, nil
}
