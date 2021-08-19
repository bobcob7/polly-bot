package internal

import (
	"database/sql"
)

type commander struct {
	db *sql.DB
}

// Help prints the brief usage of this application
func (c *commander) Help() string {
	return `Hello! I'm here to assist you captain.
If you need help with any command just add "help" in as the first argument. Ex /add-agent help
Or you can try appending "help" to the command. Ex /help agent
Here's a list of commands that are available to you:
- add-agent
- list-agents
- delete-agent
- add-subject
- list-subjects
- delete-subject
`
}
