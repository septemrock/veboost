package cmd

/*
	Command: convert
	Description: Converts a JSON file with voting power data to a CSV file.
	The JSON file should be an array of epoch data, where each epoch has a list of votes with "voter" and "votingPower".
	Usage: @<bot_name> convert --epoch <epoch_number> --total_amount <amount> [--threshold <value>] (with a JSON file attached)
	Example: @bot convert --epoch 1 --total_amount 10000 --threshold 0.001

	Flags:
	  --epoch <number>: (Required) The epoch to process from the JSON file.
	  --total_amount <number>: (Required) The total amount to distribute among the voters.
	  --threshold <float>: (Optional) The minimum proportion a voter must have to be included in the output.
*/

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"math/big"
	"strings"

	"github.com/slack-go/slack"
)

// NewConvertCommand creates the convert command.
func NewConvertCommand() Command {
	return Command{
		Name:        "convert",
		Description: "Converts a JSON file with voting power data to a CSV file.",
		Usage: `Converts a JSON file to a CSV file with addresses and allocated amounts.

*Flags:*
  --total_amount <number>: (Required) The total amount to distribute.
  --epoch <number>: (Required) The epoch to process.
  --threshold <float>: (Optional) The minimum proportion (e.g., 0.0001) a voter must have to be included.`,
		Execute: runConvertCommand,
	}
}

func runConvertCommand(ctx CommandContext) (string, error) {
	// 1. Create a new FlagSet for this command to avoid conflicts.
	csvConvertFlagSet := flag.NewFlagSet("convert", flag.ContinueOnError)

	// 2. Define the flags.
	totalAmountStr := csvConvertFlagSet.String("total_amount", "", "The total amount to be distributed (required).")
	epoch := csvConvertFlagSet.Int("epoch", -1, "The epoch to process (required).")
	threshold := csvConvertFlagSet.Float64("threshold", 0.0, "The minimum allocated amount a voter must have to be included.")

	// 3. Parse the arguments from the context.
	if err := csvConvertFlagSet.Parse(ctx.Args); err != nil {
		return "", fmt.Errorf("error parsing flags. Use `help convert` for usage details")
	}

	// 4. Validate required flags.
	if *totalAmountStr == "" {
		return "", fmt.Errorf("the --total_amount flag is required. Use `help convert` for usage details")
	}
	if *epoch == -1 {
		return "", fmt.Errorf("the --epoch flag is required. Use `help convert` for usage details")
	}

	// 5. Convert and validate the values.
	totalAmount, ok := new(big.Int).SetString(*totalAmountStr, 10)
	if !ok {
		return "", fmt.Errorf("invalid value for --total_amount. It must be a valid integer")
	}

	if len(ctx.FullMessage.Files) == 0 {
		return "", fmt.Errorf("you must attach a JSON file to use this command")
	}

	var jsonFile *slack.File
	for _, file := range ctx.FullMessage.Files {
		if strings.HasSuffix(strings.ToLower(file.Name), ".json") {
			jsonFile = &file
			break
		}
	}

	if jsonFile == nil {
		return "", fmt.Errorf("no .json file found in the attachments")
	}

	// Use a goroutine for the heavy lifting.
	go processAndUpload(ctx, *jsonFile, totalAmount, *threshold, *epoch)

	return "Request received. I'm processing the file now and will post the results shortly.", nil
}

// processAndUpload handles the downloading, processing, and uploading.
func processAndUpload(ctx CommandContext, file slack.File, totalAmount *big.Int, threshold float64, epoch int) {
	channelID := ctx.AppMentionEvent.Channel
	timestamp := ctx.AppMentionEvent.TimeStamp

	// 1. Download file content
	body, err := downloadFile(ctx, file.URLPrivateDownload)
	if err != nil {
		slog.Error("Error downloading file", "error", err, "file_url", file.URLPrivateDownload)
		postMessage(ctx.Client, channelID, "Sorry, I couldn't download the file.", timestamp, MessageTypeError)
		return
	}

	// 2. Parse JSON and calculate proportions
	voterPowers, totalVotingPower, err := parseAndCalculate(body, epoch)
	if err != nil {
		slog.Error("Error processing JSON data", "error", err)
		postMessage(ctx.Client, channelID, fmt.Sprintf("Error processing data: %v", err), timestamp, MessageTypeError)
		return
	}

	// 3. Generate CSV in memory
	csvFileName := fmt.Sprintf("converted_%s.csv", strings.ReplaceAll(timestamp, ".", "_"))
	csvBuffer, addressCount, totalDistributed, err := generateCSVBuffer(voterPowers, totalVotingPower, totalAmount, threshold)
	if err != nil {
		slog.Error("Error generating CSV", "error", err)
		postMessage(ctx.Client, channelID, "I had trouble generating the results file.", timestamp, MessageTypeError)
		return
	}

	// 4. Upload CSV to Slack
	if err := uploadFileToSlack(
		ctx,
		channelID,
		timestamp,
		csvFileName,
		"CSV Conversion Result",
		"Here are the results of the conversion:",
		csvBuffer,
	); err != nil {
		postMessage(ctx.Client, channelID, "I finished the calculations, but couldn't upload the results file.", timestamp, MessageTypeError)
	} else {
		slog.Info("Successfully uploaded CSV", "filename", csvFileName, "channel", channelID)

		// Post a summary message
		divisor := new(big.Float).SetPrec(256).SetFloat64(1e18)
		totalDistributedFloat := new(big.Float).SetPrec(256).SetInt(totalDistributed)
		totalDistributedEther := new(big.Float).Quo(totalDistributedFloat, divisor)

		// Format the amount for better readability
		formattedAmount := totalDistributedEther.Text('f', 18)
		formattedAmount = strings.TrimRight(formattedAmount, "0")
		formattedAmount = strings.TrimRight(formattedAmount, ".")

		summaryMessage := fmt.Sprintf("Conversion Summary. Total Addresses: %d. Total Amount Distributed: %s.",
			addressCount,
			formattedAmount,
		)
		postMessage(ctx.Client, channelID, summaryMessage, timestamp, MessageTypeInfo)
	}
}

func parseAndCalculate(body []byte, epochFilter int) (map[string]*big.Int, *big.Int, error) {
	type Vote struct {
		Voter       string `json:"voter"`
		VotingPower string `json:"votingPower"`
	}
	type EpochData struct {
		Epoch int    `json:"epoch"`
		Votes []Vote `json:"votes"`
	}
	var data []EpochData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, nil, fmt.Errorf("the file doesn't seem to be in the correct JSON format: %w", err)
	}

	totalVotingPower := new(big.Int)
	voterPowers := make(map[string]*big.Int)

	var epochFound bool
	for _, epoch := range data {
		if epoch.Epoch != epochFilter {
			continue
		}
		epochFound = true
		for _, vote := range epoch.Votes {
			power, ok := new(big.Int).SetString(vote.VotingPower, 10)
			if !ok {
				slog.Error("Could not parse votingPower", slog.String("votingPower", vote.VotingPower), slog.String("voter", vote.Voter))
				continue
			}
			if _, exists := voterPowers[vote.Voter]; !exists {
				voterPowers[vote.Voter] = new(big.Int)
			}
			voterPowers[vote.Voter].Add(voterPowers[vote.Voter], power)
			totalVotingPower.Add(totalVotingPower, power)
		}
		break // Found the epoch, no need to iterate further
	}

	if !epochFound {
		return nil, nil, fmt.Errorf("epoch %d not found in the provided file", epochFilter)
	}

	if totalVotingPower.Cmp(big.NewInt(0)) == 0 {
		return nil, nil, fmt.Errorf("there is no voting power in the provided file for epoch %d", epochFilter)
	}

	return voterPowers, totalVotingPower, nil
}

func generateCSVBuffer(voterPowers map[string]*big.Int, totalVotingPower, totalAmount *big.Int, threshold float64) (*bytes.Buffer, int, *big.Int, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	headers := []string{"address", "amount"}
	if err := writer.Write(headers); err != nil {
		return nil, 0, nil, fmt.Errorf("error writing CSV header: %w", err)
	}

	// Use a high precision for calculations to avoid rounding errors.
	const precision = 256
	totalPowerFloat := new(big.Float).SetPrec(precision).SetInt(totalVotingPower)
	totalAmountFloat := new(big.Float).SetPrec(precision).SetInt(totalAmount)
	thresholdFloat := new(big.Float).SetPrec(precision).SetFloat64(threshold)

	// Define the multiplier for 10^18
	multiplier := new(big.Float).SetPrec(precision).SetFloat64(1e18)

	addressCount := 0
	totalDistributedAmount := new(big.Int)

	for voter, power := range voterPowers {
		powerFloat := new(big.Float).SetPrec(precision).SetInt(power)

		// Calculate allocated amount using floating point arithmetic for precision
		allocatedAmount := new(big.Float).Quo(powerFloat, totalPowerFloat)
		allocatedAmount.Mul(allocatedAmount, totalAmountFloat)

		// If a threshold is set, filter out voters below it.
		if threshold > 0 && allocatedAmount.Cmp(thresholdFloat) < 0 {
			continue // Skip this voter
		}

		// Multiply allocatedAmount by 10^18
		allocatedAmount.Mul(allocatedAmount, multiplier)

		// Convert allocatedAmount to a big.Int
		allocatedAmountInt, _ := allocatedAmount.Int(nil) // The second return value is for inexact, which we can ignore for this purpose

		addressCount++
		totalDistributedAmount.Add(totalDistributedAmount, allocatedAmountInt)

		row := []string{
			voter,
			allocatedAmountInt.String(), // Output as integer string
		}
		if err := writer.Write(row); err != nil {
			return nil, 0, nil, fmt.Errorf("error writing row for %s: %w", voter, err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, 0, nil, fmt.Errorf("csv writer error: %w", err)
	}

	return &buffer, addressCount, totalDistributedAmount, nil
}
