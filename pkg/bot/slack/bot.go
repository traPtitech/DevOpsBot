package slack

import (
	"context"
	"fmt"
	"github.com/kballard/go-shellquote"
	"github.com/samber/lo"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"github.com/traPtitech/DevOpsBot/pkg/config"
	"github.com/traPtitech/DevOpsBot/pkg/domain"
	"go.uber.org/zap"
	"log/slog"
	"regexp"
	"strings"
)

const slashPrefix = "/"

type slackBot struct {
	api     *slack.Client
	sock    *socketmode.Client
	rootCmd domain.Command
	logger  *zap.Logger
}

func NewBot(rootCmd domain.Command, logger *zap.Logger) (domain.Bot, error) {
	// Prepare socket mode bot
	api := slack.New(config.C.Slack.OAuthToken, slack.OptionAppLevelToken(config.C.Slack.AppToken))
	sock := socketmode.New(api)

	return &slackBot{
		api:     api,
		sock:    sock,
		rootCmd: rootCmd,
		logger:  logger,
	}, nil
}

func (s *slackBot) Start(ctx context.Context) error {
	go func() {
		for e := range s.sock.Events {
			err := s.handle(e)
			if err != nil {
				s.logger.Error("failed to process event", zap.Error(err))
			}
		}
	}()
	return s.sock.RunContext(ctx)
}

func (s *slackBot) handle(e socketmode.Event) error {
	switch e.Type {
	case socketmode.EventTypeConnecting:
		s.logger.Info("Connecting to Slack with Socket Mode...")

	case socketmode.EventTypeConnectionError:
		s.logger.Info("Connection failed. Retrying later...")

	case socketmode.EventTypeConnected:
		s.logger.Info("Connected to Slack with Socket Mode.")

	case socketmode.EventTypeEventsAPI:
		eventsE, ok := e.Data.(slackevents.EventsAPIEvent)
		if !ok {
			return fmt.Errorf("failed to parse events api type")
		}

		// Acknowledge the event
		s.sock.Ack(*e.Request)

		// Process the event
		err := s.handleEventsAPI(&eventsE)
		if err != nil {
			return fmt.Errorf("failed to process events api event: %w", err)
		}

	case socketmode.EventTypeSlashCommand:
		slashE, ok := e.Data.(slack.SlashCommand)
		if !ok {
			return fmt.Errorf("failed to parse slash command type")
		}

		// Acknowledge the event
		s.sock.Ack(*e.Request, map[string]any{
			"response_type": "in_channel",
		})

		// Process the event
		err := s.handleSlashEvent(&slashE)
		if err != nil {
			return fmt.Errorf("failed to process slash event: %w", err)
		}
	}

	return nil
}

var mentionRegexp = regexp.MustCompile("^<@(\\w+)(?:\\|\\w+)?>")

func (s *slackBot) getExecutorID(ev *slackevents.MessageEvent) (executorID string, commandText string, ok bool) {
	executorID = ev.User
	commandText = ev.Text

	if ev.BotID == "" {
		// Normal execution by user
		return executorID, commandText, true
	}

	// Execution by bots - check if they are the trusted workflow members
	executorID = ev.BotID
	mentionIndices := mentionRegexp.FindStringSubmatchIndex(commandText)
	if !lo.Contains(config.C.Slack.TrustedWorkflows, executorID) {
		// If they are not trusted, ignore bots
		if mentionIndices != nil {
			// Log bot ID as they are difficult to get from UI
			slog.Info("Skipping impersonation request from bot", "bot_id", executorID, "display_name", ev.Username)
		}
		return "", "", false
	}

	// Check if the workflow is impersonating execution user
	if mentionIndices != nil && mentionIndices[0] == 0 {
		// Impersonate user
		executorID = commandText[mentionIndices[2]:mentionIndices[3]]
		// Trim the mention part
		commandText = commandText[mentionIndices[1]:]
		commandText = strings.TrimSpace(commandText)
		return executorID, commandText, true
	} else {
		// If they are not impersonating, fallback the executor to its own ID
		return executorID, commandText, true
	}
}

func (s *slackBot) handleEventsAPI(e *slackevents.EventsAPIEvent) error {
	switch ev := e.InnerEvent.Data.(type) {
	case *slackevents.MessageEvent:
		// Validate command execution context
		executorID, commandText, ok := s.getExecutorID(ev)
		if !ok {
			return nil // Not a valid user
		}
		if ev.Channel != config.C.Slack.ChannelID {
			return nil // Ignore messages not from the specified channel
		}
		if !strings.HasPrefix(commandText, config.C.Prefix) {
			return nil // Command prefix does not match
		}

		// Execute
		messageRef := slack.ItemRef{
			Channel:   ev.Channel,
			Timestamp: ev.TimeStamp,
		}
		commandText = strings.Trim(commandText, config.C.Prefix)
		return s.executeCommand(commandText, messageRef, executorID)
	default:
		return nil
	}
}

func (s *slackBot) handleSlashEvent(e *slack.SlashCommand) error {
	// Validate command execution context
	if e.ChannelID != config.C.Slack.ChannelID {
		return nil // Ignore messages not from the specified channel
	}

	// Prepare a new message to add reaction to
	commandText := fmt.Sprintf("%s %s", e.Command, e.Text)
	responseText := fmt.Sprintf("%s (<@%s|%s>) used slash command: %s",
		e.UserName, e.UserID, e.UserName,
		commandText)
	_, ts, err := s.sock.PostMessage(e.ChannelID, slack.MsgOptionText(responseText, false))
	if err != nil {
		return fmt.Errorf("failed to post message in response to slash command: %w", err)
	}

	// Execute
	messageRef := slack.ItemRef{
		Channel:   e.ChannelID,
		Timestamp: ts,
	}
	commandText = strings.TrimPrefix(commandText, slashPrefix)
	return s.executeCommand(commandText, messageRef, e.UserID)
}

func (s *slackBot) executeCommand(commandText string, messageRef slack.ItemRef, executorID string) error {
	// Prepare command args
	ctx := &slackContext{
		Context:    context.Background(),
		api:        s.api,
		logger:     s.logger,
		message:    messageRef,
		executorID: executorID,
		args:       nil,
	}
	args, err := shellquote.Split(commandText)
	if err != nil {
		return ctx.ReplyBad(fmt.Sprintf("failed to parse arguments: %v", err))
	}
	ctx.args = args

	// Execute
	return s.rootCmd.Execute(ctx)
}
