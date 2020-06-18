package tool

import (
	"github.com/slack-go/slack"
	"os"
)

var (
	SLACK_TOKEN 	= "SLACK_TOKEN"
	SLACK_CHANNEL	= "SLACK_CHANNEL"

	colorMapping 	= map[string]string{
		"prod": "#ff0000",
		"stage": "#00ff00",
		"qa": "#ffd700",
		"dev": "#663399",
		"load": "#1e90ff",
		"beta"    : "#00ff00",
	}
)

type Slack struct {
	Client 		*slack.Client
	Token 		string
	ChannelId 	string
}

func NewSlackClient() Slack {
	return Slack{
		Client: 	slack.New(os.Getenv(SLACK_TOKEN)),
		Token: 		os.Getenv(SLACK_TOKEN),
		ChannelId:  os.Getenv(SLACK_CHANNEL),
	}
}

type SlackBody struct {
	Text string `json:"text"`
}

func (s Slack) SendSimpleMessage(message string, env string) {
	if ! s.ValidClient() {
		return
	}
	color := colorMapping[env]
	attachment := slack.Attachment{
		Text: message,
		Color: color,
	}
	msgOpt := slack.MsgOptionAttachments(attachment)
	s.SendMessage(msgOpt)
}

func (s Slack) SendMessage(msgOpt slack.MsgOption) (string, string, string) {
	channel, timestamp, response, err:= s.Client.SendMessage(s.ChannelId, msgOpt)
	if err != nil {
		os.Exit(1)
	}

	return channel, timestamp, response
}

func (s Slack) CreateSimpleSection(text string) *slack.SectionBlock {
	txt := slack.NewTextBlockObject("mrkdwn", text, false, false)
	section := slack.NewSectionBlock(txt, nil,nil)
	return section
}

func (s Slack) CreateDividerSection() *slack.DividerBlock {
	return slack.NewDividerBlock()
}

func (s Slack) CreateSimpleAttachments(title, text string) slack.MsgOption {
	return slack.MsgOptionAttachments(
		slack.Attachment{
			Color:         "#36a64f",
			Title:         title,
			Text:          text,
			MarkdownIn:    []string{"text"},
		},
	)
}

//ValidClient validates slack variables
func (s Slack) ValidClient() bool {
	if len(s.Token) == 0 || len(s.ChannelId) == 0 {
		return false
	}

	return true
}
