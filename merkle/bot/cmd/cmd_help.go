package cmd

/*
	Command: help
	Description: Shows a list of all available commands or detailed help for a specific command.
	Usage: @<bot_name> help [command_name]
	Examples:
	  @bot help
	  @bot help export_airdrop
*/

import (
	"fmt"
	"strings"
)

// NewHelpCommand creates the help command.
func NewHelpCommand() Command {
	return Command{
		Name:        "help",
		Description: "Shows a list of all available commands and their descriptions.",
		Execute:     runHelpCommand,
	}
}

func runHelpCommand(ctx CommandContext) (string, error) {
	// If no arguments are provided, list all commands.
	if len(ctx.Args) == 0 {
		var builder strings.Builder
		builder.WriteString("Here are the available commands:\n")

		for _, cmd := range GetRegistry() {
			builder.WriteString(fmt.Sprintf("• `%s`: %s\n", cmd.Name, cmd.Description))
		}
		builder.WriteString("\nUse `help <command_name>` to get more information about a specific command.")
		return builder.String(), nil
	}

	// If an argument is provided, show detailed help for that specific command.
	commandName := ctx.Args[0]
	cmd, found := Get(commandName)
	if !found {
		return "", fmt.Errorf("command `%s` not found", commandName)
	}

	// Default to description if a detailed usage string is not available.
	details := cmd.Usage
	if details == "" {
		details = cmd.Description
	}

	return fmt.Sprintf("Usage: `%s`\n%s", cmd.Name, details), nil
}