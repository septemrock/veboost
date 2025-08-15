package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"

	"github.com/Bedrock-Technology/VeMerkle/internal/config"
	"github.com/Bedrock-Technology/VeMerkle/internal/contracts"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const brTokenAddressHex = "0xFf7d6A96ae471BbCD7713aF9CB1fEeB16cf56B41"

// NewAirdropInfoCommand creates the airdrop_info command.
func NewAirdropInfoCommand() Command {
	return Command{
		Name:        "airdrop_info",
		Description: "Shows the airdrop information.",
		Execute:     runAirdropInfoCommand,
	}
}

func runAirdropInfoCommand(ctx CommandContext) (string, error) {
	go processAirdropInfo(ctx)
	return "Request received. I'm fetching the airdrop information now and will post the results shortly.", nil
}

func processAirdropInfo(ctx CommandContext) {
	channelID := ctx.AppMentionEvent.Channel
	timestamp := ctx.AppMentionEvent.TimeStamp

	proxy := contracts.GetProxy()

	// 1. Get current epoch
	currentEpoch, err := proxy.GetCurrentEpoch()
	if err != nil {
		slog.Error("Error getting current epoch", "error", err)
		postMessage(ctx.Client, channelID, fmt.Sprintf("I had trouble getting the current epoch: %v", err), timestamp, MessageTypeError)
		return
	}

	// 2. Get airdrop active status
	isActive, err := proxy.IsCurrentEpochActive()
	if err != nil {
		slog.Error("Error getting airdrop status", "error", err)
		postMessage(ctx.Client, channelID, fmt.Sprintf("I had trouble getting the airdrop status: %v", err), timestamp, MessageTypeError)
		return
	}

	// 3. Get BR token balance
	rpcURL := config.GetConfig().Contracts.Airdrop.RPC
	airdropContractAddressHex := config.GetConfig().Contracts.Airdrop.Address

	balance, err := getERC20TokenBalance(rpcURL, brTokenAddressHex, airdropContractAddressHex)
	if err != nil {
		slog.Error("Error getting token balance", "error", err)
		postMessage(ctx.Client, channelID, fmt.Sprintf("I had trouble getting the token balance: %v", err), timestamp, MessageTypeError)
		return
	}

	// Format the balance from Wei to Ether
	balanceInEther := new(big.Float).SetInt(balance)
	balanceInEther.Quo(balanceInEther, new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)))

	// Build the response message
	message := fmt.Sprintf(
		"Airdrop Information:\n"+
			"  - Current Epoch: `%d`\n"+
			"  - Airdrop Active: `%t`\n"+
			"  - BR Token Balance: `%s`\n"+
			"  - Airdrop Contract Address: `%s`",
		currentEpoch,
		isActive,
		balanceInEther.Text('f', 18),
		airdropContractAddressHex,
	)

	postMessage(ctx.Client, channelID, message, timestamp, MessageTypeInfo)
}

func getERC20TokenBalance(rpcURL, tokenAddressHex, walletAddressHex string) (*big.Int, error) {
	// 1. Create an RPC client.
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC node: %v", err)
	}
	defer client.Close()

	// 2. Prepare addresses.

	tokenAddress := common.HexToAddress(tokenAddressHex)
	walletAddress := common.HexToAddress(walletAddressHex)

	// 3. Define and parse the ABI for the ERC-20 `balanceOf` function.
	// This is the JSON ABI for the `balanceOf(address)` function in the ERC-20 standard.
	const erc20ABI = `[{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}

	// 4. Pack the data required to call the `balanceOf` function.
	// The first argument is the function name, and the second is the function's parameter (the wallet address to query).
	data, err := parsedABI.Pack("balanceOf", walletAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %v", err)
	}

	// 5. Execute a read-only contract call (eth_call).
	// Since this is a read-only operation, it doesn't require sending a transaction or consuming gas.
	callMsg := ethereum.CallMsg{
		To:   &tokenAddress, // Token contract address
		Data: data,
	}
	result, err := client.CallContract(context.Background(), callMsg, nil) // nil for the latest block
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %v", err)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("contract call returned empty value, please check if the token address '%s' is correct and exists on the network", tokenAddressHex)
	}

	// 6. Parse the returned result.
	var balance *big.Int
	err = parsedABI.UnpackIntoInterface(&balance, "balanceOf", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack returned result: %v", err)
	}

	return balance, nil
}
