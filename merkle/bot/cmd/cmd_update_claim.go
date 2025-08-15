package cmd

/*
	Command: update-claim
	Description: Updates the claim status for all users for a given epoch by checking the smart contract.
	Usage: @<bot_name> update-claim --epoch <epoch_number>
	Example: @bot update-claim --epoch 1

	Flags:
	  --epoch <number>: (Required) The epoch to update the claim status for.
*/

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"math/big"

	"github.com/Bedrock-Technology/VeMerkle/internal/contracts"
	"github.com/Bedrock-Technology/VeMerkle/internal/database"
	"github.com/ethereum/go-ethereum/common"
)

func NewUpdateClaimCommand() Command {
	return Command{
		Name:        "update-claim",
		Description: "Updates the claim status for a given epoch.",
		Usage: `Updates the claim status for a given epoch.

*Flags:*
  --epoch <number>: (Required) The epoch to process.`,
		Execute: runUpdateClaimCommand,
	}
}

func runUpdateClaimCommand(ctx CommandContext) (string, error) {
	flagSet := flag.NewFlagSet("update-claim", flag.ContinueOnError)
	epoch := flagSet.Uint64("epoch", math.MaxUint64, "The epoch to process (required).")

	if err := flagSet.Parse(ctx.Args); err != nil {
		return "", fmt.Errorf("error parsing flags. Use 'help update-claim' for usage details")
	}

	if *epoch == math.MaxUint64 {
		return "", fmt.Errorf("the --epoch flag is required. Use 'help update-claim' for usage details")
	}

	go processUpdateClaim(ctx, *epoch)

	return "Request received. I'm processing the claim update now and will post the results shortly.", nil
}

func processUpdateClaim(ctx CommandContext, epoch uint64) error {
	channelID := ctx.AppMentionEvent.Channel
	timestamp := ctx.AppMentionEvent.TimeStamp

	proxy := contracts.GetProxy()

	// Step 1: Check if the current epoch's airdrop is active
	isActive, err := proxy.IsCurrentEpochActive()
	if err != nil {
		slog.Error("Failed to check if current epoch is active", slog.Any("error", err))
		postMessage(ctx.Client, channelID, "Failed to check if current epoch is active.", timestamp, MessageTypeError)
		return err
	}

	if isActive {
		postMessage(ctx.Client, channelID, fmt.Sprintf("Current epoch: `%d` is still active.", epoch), timestamp, MessageTypeError)
		return errors.New("current epoch is still active")
	}

	// Step 2: Check if the provided epoch matches the current epoch
	currentEpoch, err := proxy.GetCurrentEpoch()
	if err != nil {
		slog.Error("Failed to get current epoch", slog.Any("error", err))
		postMessage(ctx.Client, channelID, "Failed to get current epoch.", timestamp, MessageTypeError)
		return err
	}

	if epoch != currentEpoch {
		postMessage(ctx.Client, channelID, fmt.Sprintf("Provided epoch %d does not match current epoch %d.", epoch, currentEpoch), timestamp, MessageTypeError)
		return errors.New("provided epoch does not match current epoch")
	}

	// Step 3: Retrieve all users for the current epoch from the database
	userStrings, err := database.GetUsersByEpoch(epoch)
	if err != nil {
		slog.Error("Failed to get users from database", slog.Any("error", err), slog.Uint64("epoch", epoch))
		postMessage(ctx.Client, channelID, fmt.Sprintf("Failed to get users for epoch %d: %v", epoch, err), timestamp, MessageTypeError)
		return err
	}

	if len(userStrings) == 0 {
		postMessage(ctx.Client, channelID, fmt.Sprintf("No users found for epoch %d.", epoch), timestamp, MessageTypeError)
		return errors.New("no users found for epoch")
	}

	// Convert user strings to common.Address
	users := make([]common.Address, len(userStrings))
	for i, user := range userStrings {
		users[i] = common.HexToAddress(user)
	}

	// Step 4: Check claim status using the contract and update the database in batches
	batchSize := 1000
	for i := 0; i < len(users); i += batchSize {
		end := i + batchSize
		if end > len(users) {
			end = len(users)
		}

		slog.Info("Processing batch", slog.Int("batch_start", i), slog.Int("batch_end", end), slog.Int("batch_size", end-i))
		claimedStatus, err := proxy.HasUsersClaimed(big.NewInt(int64(epoch)), users[i:end])
		if err != nil {
			slog.Error("Failed to check claim status", slog.Any("error", err))
			postMessage(ctx.Client, channelID, "Failed to check claim status.", timestamp, MessageTypeError)
			return err
		}

		if err := database.UpdateClaimedStatus(epoch, userStrings[i:end], claimedStatus); err != nil {
			slog.Error("Failed to update claimed status in database", slog.Any("error", err))
			postMessage(ctx.Client, channelID, "Failed to update claimed status in database.", timestamp, MessageTypeError)
			return err
		}
		slog.Info("Batch processed successfully", "batch_start", i, "batch_end", end)
	}

	postMessage(ctx.Client, channelID, fmt.Sprintf("Successfully updated claim status for epoch %d.", epoch), timestamp, MessageTypeInfo)
	return nil
}
