package cmd

/*
	Command: merkletree
	Description: Retrieves Merkle tree information for a given epoch, such as the root, a proof for an address, or deletes the tree.
	Usage: @<bot_name> merkletree --epoch <epoch_number> [--address <address>] [--delete]
	Examples:
	  @bot merkletree --epoch 1
	  @bot merkletree --epoch 1 --address 0x1234567890123456789012345678901234567890
	  @bot merkletree --epoch 1 --delete

	Flags:
	  --epoch <number>: (Required) The epoch to process.
	  --address <string>: (Optional) The address to generate a proof for.
	  --delete: (Optional) If provided, deletes the Merkle tree for the given epoch.
*/

import (
	"flag"
	"fmt"
	"log/slog"
	"math"
	"strings"

	merkletree "github.com/FantasyJony/openzeppelin-merkle-tree-go/standard_merkle_tree"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// NewMerkleTreeCommand creates the merkletree command.
func NewMerkleTreeCommand() Command {
	return Command{
		Name:        "merkletree",
		Description: "Retrieves the Merkle root or generates a proof for a given epoch.",
		Usage: `Retrieves the Merkle root or generates a proof for a given epoch.

*Flags:*
  --epoch <number>: (Required) The epoch to process.
  --address <string>: (Optional) The address to generate a proof for.
  --delete <boolean>: (Optional) Delete the merkle tree for the given epoch.`,
		Execute: runMerkleTreeInfoCommand,
	}
}

func runMerkleTreeInfoCommand(ctx CommandContext) (string, error) {
	// 1. Create a new FlagSet.
	flagSet := flag.NewFlagSet("merkletree", flag.ContinueOnError)

	// 2. Define the flags.
	epoch := flagSet.Uint64("epoch", math.MaxUint64, "The epoch to process (required).")
	address := flagSet.String("address", "", "The address to generate a proof for (optional).")
	delete := flagSet.Bool("delete", false, "Delete the merkle tree for the given epoch.")

	// 3. Parse the arguments.
	if err := flagSet.Parse(ctx.Args); err != nil {
		return "", fmt.Errorf("error parsing flags. Use `help merkletree` for usage details")
	}

	// 4. Validate the flags.
	if *epoch == math.MaxUint64 {
		return "", fmt.Errorf("the --epoch flag is required. Use `help merkletree` for usage details")
	}

	// Use a goroutine for the heavy lifting.
	go processMerkleTreeInfo(ctx, *epoch, *address, *delete)

	return "Request received. I'm processing the request now and will post the results shortly.", nil
}

func processMerkleTreeInfo(ctx CommandContext, epoch uint64, address string, deleteTree bool) {
	channelID := ctx.AppMentionEvent.Channel
	timestamp := ctx.AppMentionEvent.TimeStamp

	if deleteTree {
		err := MerkleTreeManager.Delete(epoch)
		if err != nil {
			slog.Error("Failed to delete Merkle tree", "error", err, "epoch", epoch)
			postMessage(ctx.Client, channelID, fmt.Sprintf("Failed to delete Merkle tree for epoch %d: %v", epoch, err), timestamp, MessageTypeError)
			return
		}
		postMessage(ctx.Client, channelID, fmt.Sprintf("Successfully deleted Merkle tree for epoch %d.", epoch), timestamp, MessageTypeInfo)
		return
	}

	tree, err := MerkleTreeManager.Get(epoch)
	if err != nil {
		slog.Error("Failed to get Merkle tree", "error", err, "epoch", epoch)
		postMessage(ctx.Client, channelID, fmt.Sprintf("Failed to get Merkle tree for epoch %d: %v", epoch, err), timestamp, MessageTypeError)
		return
	}

	if address == "" {
		// Return the Merkle root
		merkleRoot := tree.Tree.GetRoot()
		hexMerkleRoot := hexutil.Encode(merkleRoot)
		postMessage(ctx.Client, channelID, fmt.Sprintf("Merkle root for epoch %d: %s", epoch, hexMerkleRoot), timestamp, MessageTypeInfo)
		return
	}

	// Generate and return the Merkle proof
	lowercaseAddress := strings.ToLower(address)
	if !common.IsHexAddress(lowercaseAddress) {
		postMessage(ctx.Client, channelID, fmt.Sprintf("Invalid address format: %s", address), timestamp, MessageTypeError)
		return
	}

	_, ok := tree.Address[lowercaseAddress]
	if !ok {
		postMessage(ctx.Client, channelID, fmt.Sprintf("Address %s not found in Merkle tree for epoch %d", address, epoch), timestamp, MessageTypeError)
		return
	}

	amount, ok := tree.Amount[lowercaseAddress]
	if !ok {
		postMessage(ctx.Client, channelID, fmt.Sprintf("Amount for address %s not found in Merkle tree for epoch %d", address, epoch), timestamp, MessageTypeError)
		return
	}

	leaf := []any{
		merkletree.SolAddress(address),
		merkletree.SolNumber(amount.String()),
	}

	proof, err := tree.Tree.GetProof(leaf)
	if err != nil {
		slog.Error("Failed to generate Merkle proof", "error", err, "epoch", epoch, "address", address)
		postMessage(ctx.Client, channelID, fmt.Sprintf("Failed to generate Merkle proof for address %s: %v", address, err), timestamp, MessageTypeError)
		return
	}

	verify, err := tree.Tree.Verify(proof, leaf)
	if err != nil {
		slog.Error("Failed to verify Merkle proof", "error", err, "epoch", epoch, "address", address)
		postMessage(ctx.Client, channelID, fmt.Sprintf("Failed to verify Merkle proof for address %s: %v", address, err), timestamp, MessageTypeError)
		return
	}

	if !verify {
		slog.Error("Merkle proof verification failed", "epoch", epoch, "address", address)
		postMessage(ctx.Client, channelID, fmt.Sprintf("Merkle proof verification failed for address %s", address), timestamp, MessageTypeError)
		return
	}

	merkleRoot := tree.Tree.GetRoot()
	hexMerkleRoot := hexutil.Encode(merkleRoot)

	response := fmt.Sprintf("Merkle root for epoch %d: %s\nAddress %s found in tree, verification successful.", epoch, hexMerkleRoot, address)
	postMessage(ctx.Client, channelID, response, timestamp, MessageTypeInfo)
}
