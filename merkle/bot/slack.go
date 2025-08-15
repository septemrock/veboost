package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/Bedrock-Technology/VeMerkle/bot/cmd"
	"github.com/Bedrock-Technology/VeMerkle/bot/config"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// SlackEventHandler handles all incoming Slack HTTP requests.
type SlackEventHandler struct {
	Client *slack.Client
	Config *config.Config
}

// NewSlackEventHandler creates a new SlackEventHandler.
func NewSlackEventHandler(config *config.Config) *SlackEventHandler {
	return &SlackEventHandler{
		Client: slack.New(config.BotToken),
		Config: config,
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *SlackEventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	verifier, err := slack.NewSecretsVerifier(r.Header, h.Config.SigningSecret)
	if err != nil {
		slog.Error("Error creating secrets verifier", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error reading request body", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := verifier.Write(body); err != nil {
		slog.Error("Error writing to verifier", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := verifier.Ensure(); err != nil {
		slog.Warn("Request signature verification failed", slog.Any("error", err))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		slog.Error("Error parsing event", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	slog.Info("Received Slack event", "type", eventsAPIEvent.Type)

	switch eventsAPIEvent.Type {
	case slackevents.URLVerification:
		h.handleURLVerification(w, body)
	case slackevents.CallbackEvent:
		h.handleCallbackEvent(w, &eventsAPIEvent)
	default:
		w.WriteHeader(http.StatusOK)
	}
}

// handleURLVerification handles Slack's URL verification requests.
func (h *SlackEventHandler) handleURLVerification(w http.ResponseWriter, body []byte) {
	var r slackevents.ChallengeResponse
	if err := json.Unmarshal(body, &r); err != nil {
		slog.Error("Error unmarshalling challenge response", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(r.Challenge))
	slog.Info("URL Verification successful", "challenge", r.Challenge)
}

// handleCallbackEvent handles all callback events.
func (h *SlackEventHandler) handleCallbackEvent(w http.ResponseWriter, event *slackevents.EventsAPIEvent) {
	slog.Info("Received inner event", slog.String("type", event.InnerEvent.Type))

	switch ev := event.InnerEvent.Data.(type) {
	case *slackevents.AppMentionEvent:
		h.handleAppMentionEvent(w, ev)
	case *slackevents.FileSharedEvent:
		slog.Info("File shared event received, ignoring as it's handled by AppMentionEvent", "file_id", ev.FileID)
	default:
		slog.Warn("Unsupported inner event type", slog.String("type", fmt.Sprintf("%T", ev)))
	}
	w.WriteHeader(http.StatusOK) // Ensure we respond with 200 OK.
}

// handleAppMentionEvent is the command dispatcher.
func (h *SlackEventHandler) handleAppMentionEvent(w http.ResponseWriter, ev *slackevents.AppMentionEvent) {
	slog.Info("App mention event received", slog.String("user", ev.User), slog.String("channel", ev.Channel), slog.String("text", ev.Text))

	// Get the full message details, which includes file attachments.
	fullMessage, err := h.getFullMessage(ev.Channel, ev.TimeStamp)
	if err != nil {
		slog.Error("Error getting full message for app_mention", slog.Any("error", err), slog.String("channel", ev.Channel), slog.String("timestamp", ev.TimeStamp))
		h.postMessage(ev.Channel, "Sorry, I had trouble retrieving the full message details.", ev.TimeStamp)
		return
	}

	// Clean the message text to extract the command and arguments.
	re := regexp.MustCompile(`<@.*?>`)
	cleanText := re.ReplaceAllString(ev.Text, "")
	fields := strings.Fields(strings.TrimSpace(cleanText))

	if len(fields) == 0 {
		fields = []string{"help"}
	}

	commandName := fields[0]
	args := fields[1:]

	// Look up the command in the registry.
	command, found := cmd.Get(commandName)
	if !found {
		response := fmt.Sprintf("Sorry, I don't recognize the command `%s`. Try `help` to see what I can do.", commandName)
		h.postMessage(ev.Channel, response, ev.TimeStamp)
		return
	}

	// Execute the command.
	ctx := cmd.CommandContext{
		AppMentionEvent: ev,
		Client:          h.Client,
		Config:          h.Config,
		FullMessage:     fullMessage,
		Args:            args,
	}

	responseText, err := command.Execute(ctx)
	if err != nil {
		slog.Error("Error executing command", slog.String("command", commandName), slog.Any("error", err))
		h.postMessage(ev.Channel, fmt.Sprintf("An error occurred while running `%s`: %v", commandName, err), ev.TimeStamp)
		return
	}

	if responseText != "" {
		h.postMessage(ev.Channel, responseText, ev.TimeStamp)
	}
}

// getFullMessage retrieves a single complete message based on channel and timestamp.
func (h *SlackEventHandler) getFullMessage(channelID, timestamp string) (*slack.Message, error) {
	replies, _, _, err := h.Client.GetConversationReplies(&slack.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: timestamp,
		Limit:     1,
		Inclusive: true,
	})
	if err != nil {
		return nil, fmt.Errorf("could not get conversation replies: %w", err)
	}
	if len(replies) == 0 {
		return nil, fmt.Errorf("could not find a message with the matching timestamp")
	}
	return &replies[0], nil
}

// postMessage sends a message to the specified channel.
func (h *SlackEventHandler) postMessage(channel, text, timestamp string) {
	_, _, err := h.Client.PostMessage(channel, slack.MsgOptionText(text, false), slack.MsgOptionTS(timestamp))
	if err != nil {
		slog.Error("Error sending message to channel", slog.String("channel", channel), slog.Any("error", err))
	}
}

func RegisterSlackBotCommands() {
	cmd.Register(cmd.NewHelpCommand())
	cmd.Register(cmd.NewConvertCommand())
	cmd.Register(cmd.NewImportCommand())
	cmd.Register(cmd.NewExportAirdropCommand())
	cmd.Register(cmd.NewMerkleTreeCommand())
	cmd.Register(cmd.NewRevertCommand())
	cmd.Register(cmd.NewAirdropInfoCommand())

	slog.Info("Registered commands", "count", len(cmd.GetRegistry()))
}
