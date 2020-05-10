package application

import (
	"github.com/slack-go/slack"
	"os"
)

var (
	SLACK_TOKEN="SLACK_TOKEN"
	SLACK_CHANNEL="SLACK_CHANNEL"
)
type Slack struct {
	Api *slack.Client
}

func NewSlackClient() Slack  {
	return Slack{
		Api: slack.New(os.Getenv(SLACK_TOKEN)),
	}
}
