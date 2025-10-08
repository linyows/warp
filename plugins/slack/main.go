package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lestrrat-go/slack"
	"github.com/linyows/warp"
)

type Slack struct{}

func (s *Slack) Notify(msg string) error {
	username := "Warp"
	icon := "https://github.com/linyows/warp/blob/main/misc/warp.svg"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	token := os.Getenv("SLACK_TOKEN")
	if len(token) == 0 {
		return fmt.Errorf("missing SLACK_TOKEN, please set `SLACK_TOKEN`")
	}

	channel := os.Getenv("SLACK_CHANNEL")
	if len(channel) == 0 {
		return fmt.Errorf("missing SLACK_CHANNEL, please set `SLACK_CHANNEL`")
	}

	cl := slack.New(token)
	_, err := cl.Chat().PostMessage(channel).Username(username).IconURL(icon).Text(msg).Do(ctx)
	return err
}

func (s *Slack) AfterInit() {
}

func (s *Slack) AfterComm(d *warp.AfterCommData) {
}

func (s *Slack) AfterConn(d *warp.AfterConnData) {
	err := s.Notify(fmt.Sprintf("`%s` => `%s` (%d msec)", d.MailFrom, d.MailTo, d.Elapse))
	if err != nil {
		fmt.Printf("[slack-plugin] %s\n", err)
	}
}

var Hook Slack //nolint
