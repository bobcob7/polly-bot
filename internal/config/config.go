package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/bobcob7/polly-bot/pkg/discord"
	"github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/cockroachdb"
	"github.com/upper/db/v4/adapter/postgresql"
)

func New() *Config {
	return &Config{
		Transmission: Transmission{
			Endpoint:          "https://transmission.bobcob7.com",
			DownloadDirectory: "/downloads/complete",
		},
	}
}

type Config struct {
	Database     Database
	Discord      discord.Config
	Transmission Transmission
	GRPC         GRPC `map:"GRPC"`
}

type GRPC struct {
	Address string
}

type Database struct {
	Type     string
	Address  string
	Username string
	Password string
	Database string
	SSLMode  string `map:"SSL_MODE"`
}

const (
	cockroachDBType = "cockroachdb"
	postgresDBType  = "postgres"
)

func (c Database) Valid() (errs MultiError) {
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
	case postgresDBType:
	case cockroachDBType:
	default:
		errs.Add(fmt.Sprintln("Unsupported type:", c.Type))
	}
	return
}

func (c Database) Session() (sess db.Session, err error) {
	switch c.Type {
	case postgresDBType:
		sess, err = postgresql.Open(postgresql.ConnectionURL{
			User:     c.Username,
			Password: c.Password,
			Host:     c.Address,
			Database: c.Database,
		})
	case cockroachDBType:
		sess, err = cockroachdb.Open(cockroachdb.ConnectionURL{
			User:     c.Username,
			Password: c.Password,
			Host:     c.Address,
			Database: c.Database,
		})
	default:
		return nil, unsupportedDatabaseError{Type: c.Type}
	}
	if err != nil {
		err = fmt.Errorf("failed to open connection: %w", err)
	}
	return
}

type unsupportedDatabaseError struct {
	Type string
}

func (u unsupportedDatabaseError) Error() string {
	return fmt.Sprintf("Unsupported type: %s", u.Type)
}

func (c Database) URL() (string, error) {
	var schema string
	switch c.Type {
	case postgresDBType:
		fallthrough
	case cockroachDBType:
		schema = c.Type
	default:
		return "", unexpectedTypeError{c.Type}
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

type Transmission struct {
	Endpoint          string
	DownloadDirectory string
	Scraper           TransmissionScraper
}

func (c Transmission) Valid() (errs MultiError) {
	_, err := url.Parse(c.Endpoint)
	if err != nil {
		errs.Add(fmt.Sprintf("Transmission Endpoint is invalid: %v", err))
	}
	if c.DownloadDirectory == "" {
		errs.Add("Transmission DownloadDirectory is required")
	}
	return
}

type TransmissionScraper struct {
	MinPeriod time.Duration
	MaxPeriod time.Duration
}

func (c TransmissionScraper) Valid() (errs MultiError) {
	return
}

type MultiError struct {
	e []string
}

func (e *MultiError) Add(errs ...string) {
	if e.e == nil {
		e.e = make([]string, 0)
	}
	e.e = append(e.e, errs...)
}

func (e *MultiError) Append(errs MultiError) {
	e.e = append(e.e, errs.e...)
}

func (e *MultiError) Ok() bool {
	return len(e.e) == 0
}

func (e MultiError) Error() string {
	var output string
	for _, err := range e.e {
		output += err + "\n"
	}
	return output
}

func (c Config) Valid() (errs MultiError) {
	errs.Append(c.Transmission.Valid())
	errs.Add(c.Discord.Valid()...)
	errs.Append(c.Database.Valid())
	return
}
