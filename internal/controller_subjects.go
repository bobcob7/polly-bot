package internal

import (
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

func (c *commander) SubjectHelp() string {
	return `Subject commands are used to add and delete subjects.
	Subjects are how you configure me to periodically check and download from different RSS feeds.
	This will use a previously made agent.
	I have 3 different commands:
	- add-subject [name] [query] (regex)
	- delete-subject [name]
	- list-subjects`
}

func (c *commander) CreateSubject(name, agentName, query, regex string) string {
	var err error
	var re *regexp.Regexp
	if regex != "" {
		// compile regex
		re, err = regexp.Compile(regex)
		if err != nil {
			logger.Error("failed to compile regex",
				zap.String("name", name),
				zap.String("regex", regex),
				zap.Error(err),
			)
			return "I don't understand that regex."
		}
	}
	agent, err := GetAgent(c.db, agentName)
	if err != nil {
		logger.Error("failed to find agent in db",
			zap.String("name", name),
			zap.String("agent-name", agentName),
			zap.Error(err),
		)
		return "I couldn't find an agent by that name"
	}
	err = AddSubject(c.db, name, query, re, agent)
	if err != nil {
		logger.Error("failed to add subject to db",
			zap.String("name", name),
			zap.String("query", query),
			zap.String("agent-name", agentName),
			zap.Error(err),
		)
		return "Failed to add subject"
	}
	return "Successfully created subject"
}

func (c *commander) CreateSubjectHelp() string {
	return `add-subject creates a new subject to filter and download links from an RSS feed.
	This command takes 3-4 arguments, a name, an agent name, a search string and an optional regular expression.
	The name can be anything, but must be unique.
	The agent name must match an agent that you've previously made with add-agent.
	The search string the search query that will be used to filter the RSS feed.
	The regex string is a golang formatted regular expression that will filter all feed entries by their title.`
}

func (c *commander) ListSubject() string {
	subjects, err := GetSubjects(c.db)
	if err != nil {
		logger.Error("failed to get subjects from db", zap.Error(err))
	}
	output := make([]string, 0, len(subjects))
	for _, subject := range subjects {
		newSubjectString := fmt.Sprintf("%s: %s", subject.Name, subject.GetURL())
		output = append(output, newSubjectString)
	}
	if len(output) == 0 {
		return "Couldn't find any subjects"
	}
	return strings.Join(output, "\n")
}

func (c *commander) ListSubjectHelp() string {
	return `list-subjects will list all available subjects and their information. This doesn't take any arguments.`
}

func (c *commander) DeleteSubject(name string) string {
	err := DeleteSubject(c.db, name)
	if err != nil {
		logger.Error("failed to delete subject", zap.Error(err))
		return fmt.Sprintf("Failed to delete subject %s", name)
	}
	return fmt.Sprintf("Deleted subject %s", name)
}

func (c *commander) DeleteSubjectHelp() string {
	return `delete-subjects will delete a subject. The only argument is the subject's name.`
}
