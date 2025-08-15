package cmd

import (
	"github.com/Bedrock-Technology/VeMerkle/bot/config"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// CommandContext provides the command with necessary dependencies and event data.
type CommandContext struct {
	AppMentionEvent *slackevents.AppMentionEvent
	Client          *slack.Client
	Config          *config.Config
	FullMessage     *slack.Message
	Args            []string
}

// Command defines the standard for a bot command.
type Command struct {
	Name        string
	Description string
	Usage       string // Detailed usage message including flags.
	Execute     func(ctx CommandContext) (string, error)
}

// registry holds all registered commands.
var registry = make(map[string]Command)

// Register adds a command to the registry.
func Register(cmd Command) {
	registry[cmd.Name] = cmd
}

// Get retrieves a command from the registry.
func Get(name string) (Command, bool) {
	cmd, found := registry[name]
	return cmd, found
}

// GetRegistry returns the entire command registry.
func GetRegistry() map[string]Command {
	return registry
}
