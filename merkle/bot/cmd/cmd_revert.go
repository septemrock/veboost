package cmd

import (
	"flag"
	"fmt"
	"log/slog"
	"math"
)

/*
	Command: revert
	Description: Reverts the data for a specific epoch.
	Usage: @<bot_name> revert --epoch <epoch_number>
	Examples:
	  @bot revert --epoch 1

	Flags:
	  --epoch <number>: (Required) The epoch to revert. The epoch must be currentEpoch + 1 on-chain.
*/

// NewRevertCommand creates the revert command.
func NewRevertCommand() Command {
	return Command{
		Name:        "revert",
		Description: "Reverts the data for a specific epoch.",
		Usage: `Reverts the data for a specific epoch.

*Flags:*
  --epoch <number>: (Required) The epoch to revert. The epoch must be currentEpoch + 1 on-chain.`,
		Execute: runRevertCommand,
	}
}

func runRevertCommand(ctx CommandContext) (string, error) {
	// 1. Create a new FlagSet.
	flagSet := flag.NewFlagSet("revert", flag.ContinueOnError)

	// 2. Define the flags.
	epoch := flagSet.Uint64("epoch", math.MaxUint64, "The epoch to revert (required).")

	// 3. Parse the arguments.
	if err := flagSet.Parse(ctx.Args); err != nil {
		return "", fmt.Errorf("error parsing flags. Use `help revert` for usage details")
	}

	// 4. Validate the flags.
	if *epoch == math.MaxUint64 {
		return "", fmt.Errorf("the --epoch flag is required. Use `help revert` for usage details")
	}

	// Use a goroutine for the heavy lifting.
	go processRevertCommand(ctx, *epoch)

	return "Request received. I'm processing the revert request now and will post the results shortly.", nil
}

func processRevertCommand(ctx CommandContext, epoch uint64) {
	channelID := ctx.AppMentionEvent.Channel
	timestamp := ctx.AppMentionEvent.TimeStamp

	// Delete the Merkle tree and airdrop data for the epoch.
	// The Delete function now contains the validation logic.
	if err := MerkleTreeManager.Delete(epoch); err != nil {
		msg := fmt.Sprintf("Failed to revert data for epoch %d: %v", epoch, err)
		slog.Error(msg)
		postMessage(ctx.Client, channelID, msg, timestamp, MessageTypeError)
		return
	}

	msg := fmt.Sprintf("Successfully reverted all data for epoch %d.", epoch)
	slog.Info(msg)
	postMessage(ctx.Client, channelID, msg, timestamp, MessageTypeInfo)
}
