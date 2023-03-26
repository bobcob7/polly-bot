package config

import (
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/bobcob7/polly-bot/pkg/discord"
	"github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/cockroachdb"
	"github.com/upper/db/v4/adapter/postgresql"
)

func New() *Config {
	return &Config{
		Transmission: ConfigTransmission{
			Endpoint:          "https://transmission.bobcob7.com",
			DownloadDirectory: "/downloads/complete",
		},
	}
}

type Config struct {
	RSSPeriod     string `map:"RSS_PERIOD"`
	HistoryLength int    `map:"HISTORY_LENGTH"`
	Database      ConfigDatabase
	Discord       discord.Config
	Transmission  ConfigTransmission
	GRPC          ConfigGRPC `map:"GRPC"`
}

type ConfigGRPC struct {
	Address string
}

type ConfigDatabase struct {
	Type     string
	Address  string
	Username string
	Password string
	Database string
	SSLMode  string `map:"SSL_MODE"`
}

func (c ConfigDatabase) Valid() (errs Errors) {
	if c.Address == "" {
		errs.Add("Address is required")
	}
	if c.Username == "" {
		errs.Add("Username is required")
	}
	if c.Database == "" {
		errs.Add("Database is required")
	}
	if c.Type == "" {
		errs.Add("Type is required")
	}
	switch c.Type {
	case "postgres":
	case "cockroachdb":
	default:
		errs.Add(fmt.Sprintln("Unsupported type:", c.Type))
	}
	return
}

func (c ConfigDatabase) Session() (db.Session, error) {
	switch c.Type {
	case "postgres":
		return postgresql.Open(postgresql.ConnectionURL{
			User:     c.Username,
			Password: c.Password,
			Host:     c.Address,
			Database: c.Database,
		})
	case "cockroachdb":
		return cockroachdb.Open(cockroachdb.ConnectionURL{
			User:     c.Username,
			Password: c.Password,
			Host:     c.Address,
			Database: c.Database,
		})
	}
	return nil, fmt.Errorf("Unsupported type: %s", c.Type)
}

func (c ConfigDatabase) URL() (string, error) {
	var schema string
	switch c.Type {
	case "postgres":
		schema = "postgres"
	case "cockroachdb":
		schema = "cockroachdb"
	default:
		return "", fmt.Errorf("Unsupported type: %s", c.Type)
	}
	credentials := url.PathEscape(c.Username)
	if c.Password != "" {
		credentials += ":" + url.PathEscape(c.Password)
	}
	connString := fmt.Sprintf("%s://%s@%s/%s", schema, credentials, c.Address, c.Database)
	if c.SSLMode != "" {
		connString += fmt.Sprintf("?sslmode=%s", c.SSLMode)
	}
	return connString, nil
}

type ConfigTransmission struct {
	Endpoint          string
	DownloadDirectory string
	Scraper           ConfigTransmissionScraper
}

func (c ConfigTransmission) Valid() (errs Errors) {
	_, err := url.Parse(c.Endpoint)
	if err != nil {
		errs.Add(fmt.Sprintf("Transmission Endpoint is invalid: %v", err))
	}
	if c.DownloadDirectory == "" {
		errs.Add("Transmission DownloadDirectory is required")
	}
	return
}

type ConfigTransmissionScraper struct {
	MinPeriod time.Duration
	MaxPeriod time.Duration
}

func (c ConfigTransmissionScraper) Valid() (errs Errors) {
	return
}

type Errors struct {
	e []string
}

func (e *Errors) Add(errs ...string) {
	if e.e == nil {
		e.e = make([]string, 0)
	}
	e.e = append(e.e, errs...)
}

func (e *Errors) Append(errs Errors) {
	e.e = append(e.e, errs.e...)
}

func (e *Errors) Ok() bool {
	return len(e.e) == 0
}

func (e Errors) Error() string {
	var output string
	for _, err := range e.e {
		output += err + "\n"
	}
	return output
}

var schemaRe = regexp.MustCompile(`^([A-Za-z0-9]+):\/\/.*$`)

func (c Config) Valid() (errs Errors) {
	if c.RSSPeriod == "" {
		errs.Add("RSSPeriod is required")
	} else if rssPeriod, err := time.ParseDuration(c.RSSPeriod); err != nil {
		errs.Add(fmt.Sprintf("RSSPeriod must be a valid duration: %v", err))
	} else if rssPeriod < time.Second*5 {
		errs.Add("RSSPeriod must be at least 5 seconds")
	} else if rssPeriod > time.Hour*24 {
		errs.Add("RSSPeriod must be less than 24 hours")
	}
	if c.HistoryLength < 10 {
		errs.Add("HistoryLength must be at least 10")
	} else if c.HistoryLength > 100000 {
		errs.Add("HistoryLength must be less than 100000")
	}
	errs.Append(c.Transmission.Valid())
	errs.Add(c.Discord.Valid()...)
	errs.Append(c.Database.Valid())
	return
}
