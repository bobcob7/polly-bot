package internal

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

const createAgentsTable = `CREATE TABLE IF NOT EXISTS agents (
	name STRING PRIMARY KEY,
	base_url STRING NOT NULL,
	query_formatter STRING DEFAULT '%s&q=%s'
);`

type Agent struct {
	Name           string
	baseURL        string
	queryFormatter string
}

func (a *Agent) GetBaseURL() string {
	return a.baseURL
}

func NewAgent(scanner interface {
	Scan(...interface{}) error
}) (*Agent, error) {
	var agent Agent
	err := scanner.Scan(&agent.Name, &agent.baseURL, &agent.queryFormatter)
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

func GetAgent(db *sql.DB, name string) (*Agent, error) {
	logger.Info("Getting Agent",
		zap.String("name", name),
	)
	res := db.QueryRow(`SELECT name, base_url, query_formatter FROM agents WHERE name = $1`, name)
	if res.Err() != nil {
		return nil, res.Err()
	}
	return NewAgent(res)
}

func DeleteAgent(db *sql.DB, name string) error {
	logger.Info("Deleting Agent",
		zap.String("name", name),
	)
	res, err := db.Exec(`DELETE FROM agents WHERE name = $1`, name)
	if err != nil {
		return err
	}
	i, _ := res.RowsAffected()
	if i == 0 {
		return fmt.Errorf("could not find agent")
	}
	return nil
}

func AddAgent(db *sql.DB, name, baseURL string) error {
	logger.Info("Adding Agent",
		zap.String("name", name),
		zap.String("baseURL", baseURL),
	)
	// Count duplicates
	res := db.QueryRow(`SELECT count(*) FROM agents WHERE name = $1`, name)
	if res.Err() != nil {
		return res.Err()
	}
	var count int
	if err := res.Scan(&count); err != nil {
		return err
	}
	// Insert
	_, err := db.Exec(`INSERT INTO agents (name, base_url) VALUES ($1, $2)`, name, baseURL)
	return err
}

type Agents []*Agent

func GetAgents(db *sql.DB) ([]*Agent, error) {
	logger.Info("Getting Agents")
	rows, err := db.Query(`SELECT name, base_url, query_formatter FROM agents`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	output := make([]*Agent, 0)
	for rows.Next() {
		newAgent, err := NewAgent(rows)
		if err != nil {
			return nil, err
		}
		output = append(output, newAgent)
	}
	return output, nil
}
