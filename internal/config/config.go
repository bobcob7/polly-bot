package config

import (
	"fmt"
	"net/url"
	"regexp"
	"time"
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
	Discord       ConfigDiscord
	Transmission  ConfigTransmission
}

type ConfigDatabase struct {
	ConnectionString string
}

func (c ConfigDatabase) Valid() (errs Errors) {
	if !schemaRe.MatchString(c.ConnectionString) {
		errs.Add("ConnectionString must have a valid schema")
	}
	return
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
	errs.Append(c.Discord.Valid())
	return
}

type ConfigDiscord struct {
	Token      string
	GuildID    string `map:"GUILD_ID"`
	RootUserID string `map:"ROOT_USER_ID"`
}

func (c ConfigDiscord) Valid() (errs Errors) {
	if c.Token == "" {
		errs.Add("Discord Token is required")
	}
	if c.RootUserID == "" {
		errs.Add("Discord RootUserID is required")
	}
	return
}
