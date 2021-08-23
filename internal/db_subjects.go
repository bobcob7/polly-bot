package internal

import (
	"database/sql"
	"fmt"
	"net/url"
	"regexp"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Subject struct {
	Name       string
	SearchText string
	Regex      *regexp.Regexp
	URL        string
}

func (s *Subject) GetURL() string {
	return s.URL
}
func (s *Subject) MatchString(val string) bool {
	if s.Regex == nil {
		return true
	}
	return s.Regex.MatchString(val)
}

func NewSubject(scanner interface {
	Scan(...interface{}) error
}) (*Subject, error) {
	var subject Subject
	var rawRe sql.NullString
	var baseURL, queryFormatter string
	err := scanner.Scan(&subject.Name, &subject.SearchText, &rawRe, &baseURL, &queryFormatter)
	if err != nil {
		return nil, err
	}
	if rawRe.Valid {
		subject.Regex, err = regexp.Compile(rawRe.String)
		if err != nil {
			return nil, err
		}
	}
	subject.URL = fmt.Sprintf(queryFormatter, baseURL, url.QueryEscape(subject.SearchText))
	return &subject, nil
}

func GetSubject(db *sql.DB, name string) (*Agent, error) {
	logger.Info("Getting Subject",
		zap.String("name", name),
	)
	res := db.QueryRow(`SELECT
		subjects.name,
		subjects.search_text,
		subjects.regex,
		agents.base_url,
		agents.query_formatter,
	FROM subjects INNER JOIN agents ON subjects.agent_name = agents.name WHERE name = $1`, name)
	if res.Err() != nil {
		return nil, res.Err()
	}
	return NewAgent(res)
}

func DeleteSubject(db *sql.DB, name string) error {
	logger.Info("Deleting Subject",
		zap.String("name", name),
	)
	res, err := db.Exec(`DELETE FROM subjects WHERE name = $1`, name)
	if err != nil {
		return err
	}
	i, _ := res.RowsAffected()
	if i == 0 {
		return fmt.Errorf("could not find subject")
	}
	return nil
}

func AddSubject(db *sql.DB, name, searchText string, regex *regexp.Regexp, agent *Agent) error {
	logger.Info("Adding Subject",
		zap.String("name", name),
		zap.String("searchText", searchText),
		zap.String("regex", regex.String()),
		zap.String("agent", agent.Name),
	)
	// Count duplicates
	res := db.QueryRow(`SELECT count(*) FROM subjects WHERE name = $1`, name)
	if res.Err() != nil {
		return res.Err()
	}
	var count int
	if err := res.Scan(&count); err != nil {
		return err
	}
	// Insert
	var err error
	if regex != nil {
		_, err = db.Exec(`INSERT INTO subjects (name, search_text, regex, agent_name) VALUES ($1, $2, $3, $4)`, name, searchText, regex.String(), agent.Name)
	} else {
		_, err = db.Exec(`INSERT INTO subjects (name, search_text, agent_name) VALUES ($1, $2, $3)`, name, searchText, agent.Name)
	}
	return err
}

type Subjects []*Subject

func GetSubjects(db *sql.DB) ([]*Subject, error) {
	logger.Info("Getting Subjects")
	rows, err := db.Query(`SELECT
	subjects.name,
	subjects.search_text,
	subjects.regex,
	agents.base_url,
	agents.query_formatter
FROM subjects INNER JOIN agents ON subjects.agent_name = agents.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	output := make([]*Subject, 0)
	for rows.Next() {
		newAgent, err := NewSubject(rows)
		if err != nil {
			return nil, err
		}
		output = append(output, newAgent)
	}
	return output, nil
}
