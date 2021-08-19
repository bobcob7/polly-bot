package internal

import (
	"database/sql"
	"fmt"
	"regexp"

	"go.uber.org/zap"
)

func InitDB(db *sql.DB, demo bool) error {
	logger.Info("Initializing database connection",
		zap.Bool("demo", demo),
	)
	if _, err := db.Exec(createAgentsTable); err != nil {
		return fmt.Errorf("failed to create agents table: %w", err)
	}
	if _, err := db.Exec(createSubjectsTable); err != nil {
		return fmt.Errorf("failed to create subjects table: %w", err)
	}
	if demo {
		agents, err := GetAgents(db)
		if err != nil {
			return fmt.Errorf("failed to get agents: %w", err)
		}
		if len(agents) < 1 {
			logger.Info("Adding demo agent")
			// Add agent
			err := AddAgent(db, "nyaa", "https://nyaa.si/?page=rss&c=0_0&f=0")
			if err != nil {
				return fmt.Errorf("failed to create demo agent: %w", err)
			}
		}
		agent, err := GetAgent(db, "nyaa")
		if err != nil {
			return fmt.Errorf("failed to get demo agent: %w", err)
		}
		subjects, err := GetSubjects(db)
		if err != nil {
			return fmt.Errorf("failed to get subjects: %w", err)
		}
		if len(subjects) < 1 {
			logger.Info("Adding demo subject")
			// Add agent
			err := AddSubject(db,
				"Combatants will be dispatched",
				"combatants will be dispatched dub",
				regexp.MustCompile(`\[Golumpa\].*\[English Dub\].*1080p.*\[MKV\]`),
				agent,
			)
			if err != nil {
				return fmt.Errorf("failed to create demo subject: %w", err)
			}
		}
	}
	return nil
}
