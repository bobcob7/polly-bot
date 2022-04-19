package internal

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// AgentHelp prints the usage of agents
func (c *commander) AgentHelp() string {
	return `Agent commands are used to add and delete agents.
	Agents are how you configure me to understand different RSS sites.
	I have 3 different commands:
	- add-agent [name] [base-url]
	- delete-agent [name]
	- list-agents`
}

func (c *commander) CreateAgentHelp() string {
	return `add-agent creates a new agent to poll RSS feeds with.
	This command takes 2 arguments, a name and a base-url.
	The name can be anything, but must be unique.
	The base-url is the URL including the schema that points to the main RSS endpoint. Ex "https://nyaa.si/?page=rss&f=0"`
}

func (c *commander) CreateAgent(name, url string) string {
	// At least validate that it's a valid schema
	// if !schemaRe.MatchString(url) {
	// 	logger.Error("invalid agent URL",
	// 		zap.String("name", name),
	// 		zap.String("url", url),
	// 	)
	// 	return "That URL doesn't look right."
	// }
	err := AddAgent(c.db, name, url)
	if err != nil {
		logger.Error("failed to add agent to db",
			zap.String("name", name),
			zap.String("url", url),
			zap.Error(err),
		)
		return "Failed to add agent"
	}
	return "Successfully create a new agent"
}

func (c *commander) ListAgentHelp() string {
	return `list-agents will list all available agents and their information. This doesn't take any arguments.`
}

func (c *commander) ListAgent() string {
	agents, err := GetAgents(c.db)
	if err != nil {
		logger.Error("failed to get agents from db", zap.Error(err))
	}
	output := make([]string, 0, len(agents))
	for _, agent := range agents {
		newAgentString := fmt.Sprintf("%s: %s", agent.Name, agent.GetBaseURL())
		output = append(output, newAgentString)
	}
	if len(output) == 0 {
		return "Couldn't find any agents"
	}
	return strings.Join(output, "\n")
}

func (c *commander) DeleteAgentHelp() string {
	return `delete-agent will delete an unused agent. The only argument is the agent's name.`
}

func (c *commander) DeleteAgent(name string) string {
	err := DeleteAgent(c.db, name)
	if err != nil {
		logger.Error("failed to delete agent", zap.Error(err))
		return fmt.Sprintf("Failed to delete agent %s", name)
	}
	return fmt.Sprintf("Deleted agent %s", name)
}
