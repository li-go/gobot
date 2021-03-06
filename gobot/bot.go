package gobot

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/nlopes/slack"

	"github.com/li-go/gobot/ai"
)

var (
	ErrInvalidHandler    = errors.New("invalid handler")
	ErrDuplicateRegister = errors.New("duplicate register")
)

type Bot interface {
	RegisterHandler(Handler) error
	Start()
	Stop()
	GetRTM() *slack.RTM
	GetLogger() *log.Logger
	SendMessage(string, string)
	LoadChannel(string) (string, error)
	LoadUser(string) (string, error)
	Help() string
}

type bot struct {
	rtm       *slack.RTM
	logger    *log.Logger
	msgParser *MessageParser
	user      string
	channels  map[string]string
	users     map[string]string

	handlers []Handler

	stopped bool
}

func New(token string, logger *log.Logger) (Bot, error) {
	rtm := slack.New(token).NewRTM()
	res, err := rtm.AuthTest()
	if err != nil {
		return nil, err
	}

	return &bot{
		rtm:       rtm,
		logger:    logger,
		msgParser: NewMessageParser(res.UserID),
		user:      "@" + res.User,
		channels:  make(map[string]string),
		users:     make(map[string]string),
	}, nil
}

func (bot *bot) RegisterHandler(handler Handler) error {
	if !handler.IsValid() {
		return ErrInvalidHandler
	}
	for _, h := range bot.handlers {
		if h.Name == handler.Name {
			return ErrDuplicateRegister
		}
	}
	bot.handlers = append(bot.handlers, handler)
	return nil
}

func (bot *bot) Stop() {
	bot.stopped = true
	bot.logger.Print("bot stopped")
}

func (bot *bot) Start() {
	go bot.rtm.ManageConnection()
	bot.logger.Print("start receiving incoming events...")
	for ev := range bot.rtm.IncomingEvents {
		if bot.stopped {
			break
		}

		if msg, ok := ev.Data.(*slack.MessageEvent); ok {
			bot.onMessage(msg)
		}
	}
}

func (bot *bot) onMessage(msg *slack.MessageEvent) {
	// ignore bot message
	if len(msg.BotID) > 0 {
		return
	}

	if _, err := bot.LoadChannel(msg.Channel); err != nil {
		bot.logger.Print(err)
		return
	}
	if _, err := bot.LoadUser(msg.User); err != nil {
		bot.logger.Print(err)
		return
	}

	parsedMsg := bot.msgParser.Parse(msg.Text, msg.Channel, msg.User)

	var handled bool
	for _, handler := range bot.handlers {
		if handler.NeedsMention && parsedMsg.Type == ListenTo {
			continue
		}
		if !handler.Handleable(bot, parsedMsg) {
			continue
		}
		go bot.handle(handler, parsedMsg)
		// handle message only once
		handled = true
		break
	}

	if !handled && parsedMsg.Type != ListenTo {
		bot.SendMessage(ai.Answer(msg.Text), msg.Channel)
	}
}

func (bot *bot) handle(handler Handler, msg Message) {
	if err := handler.Handle(bot, msg); err != nil {
		bot.SendMessage(fmt.Sprintf("<@%s> *failed* - `%s` :see_no_evil: (error: %s)", msg.UserID, msg.Text, err), msg.ChannelID)
	}
}

func (bot *bot) GetRTM() *slack.RTM {
	return bot.rtm
}

func (bot *bot) GetLogger() *log.Logger {
	return bot.logger
}

func (bot *bot) SendMessage(text string, channelID string) {
	bot.rtm.SendMessage(bot.rtm.NewOutgoingMessage(text, channelID))
}

func (bot *bot) LoadChannel(channelID string) (string, error) {
	if c, ok := bot.channels[channelID]; ok {
		return c, nil
	}

	c, err := bot.rtm.GetConversationInfo(channelID, false)
	if err != nil {
		return "", fmt.Errorf("fail to get connversation(%s): %v", channelID, err)
	}
	bot.channels[channelID] = "#" + c.Name
	if c.IsIM {
		bot.channels[channelID] = "<direct message>"
	}
	return bot.channels[channelID], nil
}

func (bot *bot) LoadUser(userID string) (string, error) {
	// WTF: empty userID
	if len(userID) == 0 {
		return "", nil
	}

	if u, ok := bot.users[userID]; ok {
		return u, nil
	}

	user, err := bot.rtm.GetUserInfo(userID)
	if err != nil {
		return "", fmt.Errorf("fail to get user(%s): %v", userID, err)
	}
	bot.users[userID] = "@" + user.Profile.DisplayName
	return bot.users[userID], nil
}

func (bot *bot) Help() string {
	h := []string{"```", "available commands:"}
	for _, handler := range bot.handlers {
		s := "  * "
		if handler.NeedsMention {
			s += bot.user + " "
		}
		s += handler.Help
		h = append(h, s)
	}
	h = append(h, "```")
	return strings.Join(h, "\n")
}
