package configurablecommand

import (
	"errors"
	"fmt"
	"strings"

	"github.com/li-go/gobot/cmdargparser"
	"github.com/li-go/gobot/gobot"
)

var (
	errMsgFmt = "```\nerror:\n  %s\n```\n:thinking_face:"

	ErrNoPermission = errors.New("no permission")
)

type Command struct {
	Name         string
	Command      string
	ParamNames   []string `yaml:"params"`
	LogFilename  string   `yaml:"log"`
	ErrChannelID string   `yaml:"error_channel"`
	ChannelNames []string `yaml:"channels"`
	UserNames    []string `yaml:"users"`
}

func (c Command) Handler() gobot.Handler {
	return gobot.Handler{
		Name:         c.Name,
		Help:         c.help(),
		NeedsMention: true,
		Handleable: func(bot gobot.Bot, msg gobot.Message) bool {
			m, _ := c.match(msg.Text)
			return m
		},
		Handle: func(bot gobot.Bot, msg gobot.Message) error {
			return addTask(bot, msg, c)
		},
	}
}

func (c Command) help() string {
	ss := []string{c.Name}
	for _, p := range c.ParamNames {
		ss = append(ss, fmt.Sprintf("[--%s=<%s>]", p, p))
	}
	return strings.Join(ss, " ")
}

func (c Command) newExecutor(bot gobot.Bot, msg gobot.Message) (*Executor, error) {
	_, paramString := c.match(msg.Text)
	params, err := c.parseParams(paramString)
	if err != nil {
		bot.SendMessage(fmt.Sprintf(errMsgFmt, err.Error()), msg.ChannelID)
		return nil, err
	}

	channel, err := bot.LoadChannel(msg.ChannelID)
	if err != nil {
		return nil, err
	}
	user, err := bot.LoadUser(msg.UserID)
	if err != nil {
		return nil, err
	}
	if !c.hasPermission(channel, user) {
		bot.SendMessage(fmt.Sprintf(errMsgFmt, "you are not allowed to do that"), msg.ChannelID)
		return nil, ErrNoPermission
	}

	executor, err := NewExecutor(c, params)
	if err != nil {
		return nil, err
	}

	// post_slack hook
	go func(e *Executor, msg gobot.Message) {
		for {
			slackMsg, ok := e.NextSlackMessage()
			if !ok {
				break
			}
			bot.SendMessage(slackMsg, msg.ChannelID)
		}
	}(executor, msg)

	// error message hook
	go func(e *Executor, c Command) {
		errChannelID := c.ErrChannelID
		if len(errChannelID) == 0 {
			errChannelID = msg.ChannelID
		}

		for {
			errMsg, ok := e.NextErrorMessage()
			if !ok {
				break
			}
			errMsg = strings.TrimSuffix(errMsg, "\n")
			bot.SendMessage(fmt.Sprintf(errMsgFmt, errMsg), errChannelID)
		}
	}(executor, c)

	return executor, nil
}

func (c Command) match(text string) (bool, string) {
	if !strings.HasPrefix(text, c.Name) {
		return false, ""
	}
	if len(text) == len(c.Name) {
		return true, ""
	}
	text = text[len(c.Name):]
	if text[0] != ' ' {
		return false, ""
	}
	return true, text[1:]
}

func (c Command) isValidParamName(name string) bool {
	for _, n := range c.ParamNames {
		if n == name {
			return true
		}
	}
	return false
}

type param struct {
	Name  string
	Value string
}

func (c Command) parseParams(text string) ([]param, error) {
	params, err := cmdargparser.Parse(text)
	if err != nil {
		return nil, err
	}
	var pp []param
	for _, p := range params {
		var found bool
		for _, name := range c.ParamNames {
			if p.Name == name {
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New("unknown param name")
		}
		pp = append(pp, param{Name: p.Name, Value: p.Value})
	}
	return pp, nil
}

func (c Command) hasPermission(channelName, userName string) bool {
	if len(c.ChannelNames) > 0 {
		var found bool
		for _, n := range c.ChannelNames {
			if n == channelName {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(c.UserNames) > 0 {
		var found bool
		for _, n := range c.UserNames {
			if n == userName {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
