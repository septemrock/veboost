package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/slack-go/slack"
)

// downloadFile handles the download of a private file from Slack.
// It creates an HTTP request and adds the bot's OAuth token to the
// Authorization header, which is required to access private files.
// It returns the file content as a byte slice or an error if the download fails.
func downloadFile(ctx CommandContext, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	// Add the bot token to the authorization header to download the private file.
	req.Header.Add("Authorization", "Bearer "+ctx.Config.BotToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// MessageType defines the type of message to be posted.
type MessageType string

const (
	// MessageTypeError is for error messages.
	MessageTypeError MessageType = "danger"
	// MessageTypeInfo is for informational messages.
	MessageTypeInfo MessageType = "good"
)

// postMessage is a helper function to send a message back to a Slack channel.
// It uses the provided timestamp to post the message as a reply in a thread,
// keeping the conversation organized.
func postMessage(client *slack.Client, channel, text, timestamp string, messageType MessageType) {
	attachment := slack.Attachment{
		Text:  text,
		Color: string(messageType),
	}
	_, _, err := client.PostMessage(channel, slack.MsgOptionAttachments(attachment), slack.MsgOptionTS(timestamp))
	if err != nil {
		slog.Error("Error sending message to channel", slog.String("channel", channel), slog.Any("error", err))
	}
}

// uploadFileToSlack handles uploading a file to a specified Slack channel.
// The file content is provided as a bytes.Buffer. The function allows setting
// a file name, title, and an initial comment. It can also post the file
// within a specific thread by using the thread's timestamp.
func uploadFileToSlack(ctx CommandContext, channelID, threadTs, fileName, title, comment string, fileContent *bytes.Buffer) error {
	params := slack.UploadFileV2Parameters{
		Reader:   fileContent,
		Filename: fileName,
		FileSize: fileContent.Len(),
		Channel:  channelID,
	}
	if threadTs != "" {
		params.ThreadTimestamp = threadTs
	}
	if title != "" {
		params.Title = title
	}
	if comment != "" {
		params.InitialComment = comment
	}

	_, err := ctx.Client.UploadFileV2(params)
	if err != nil {
		slog.Error("Error uploading file to Slack", "error", err, "channel", channelID)
		return err
	}
	slog.Info("Successfully uploaded file", "filename", fileName, "channel", channelID)
	return nil
}
