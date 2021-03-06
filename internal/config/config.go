package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/bobcob7/polly/pkg/discord"
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
	RSSPeriod     string         `map:"RSS_PERIOD"`
	HistoryLength int            `map:"HISTORY_LENGTH"`
	Database      ConfigDatabase `map:"DATABASE"`
	Discord       discord.Config
	Transmission  ConfigTransmission
}

type ConfigDatabase struct {
	Address          string
	Port             int
	Username         string
	Password         string
	Database         string
	SSLMode          string `map:"SSL_MODE"`
	ConnectionString string `map:"CONNECTION_STRING"`
}

func (c ConfigDatabase) Valid() (errs Errors) {
	if c.ConnectionString != "" {
		if !schemaRe.MatchString(c.ConnectionString) {
			errs.Add("ConnectionString must have a valid schema")
		}
	} else {
		if c.Address == "" {
			errs.Add("Address is required")
		}
		if c.Port <= 0 {
			errs.Add("Port must be above 0")
		}
		if c.Port > 35565 {
			errs.Add("Port must be below 35565")
		}
		if c.Username == "" {
			errs.Add("Username is required")
		}
		if c.Database == "" {
			errs.Add("Database is required")
		}
	}
	return
}

func (c ConfigDatabase) String() string {
	if c.ConnectionString != "" {
		return c.ConnectionString
	}
	credentials := c.Username
	if c.Password != "" {
		credentials += ":" + c.Password
	}
	output := fmt.Sprintf("postgres://%s@%s:%d/%s", credentials, c.Address, c.Port, c.Database)
	options := []string{}
	if c.SSLMode != "" {
		options = append(options, "sslmode="+c.SSLMode)
	}
	if len(options) > 0 {
		output = output + "?" + strings.Join(options, "&")
	}
	return output
}

type ConfigTransmission struct {
	Endpoint          string
	DownloadDirectory string
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
	return
}
