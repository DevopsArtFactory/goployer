package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/slack-go/slack"
	"net/http"
	"os"
	"time"
)

var (
	SLACK_TOKEN       = "SLACK_TOKEN"
	SLACK_CHANNEL     = "SLACK_CHANNEL"
	SLACK_WEBHOOK_URL = "SLACK_WEBHOOK_URL"

	colorMapping = map[string]string{
		"prod":  "#ff0000",
		"stage": "#00ff00",
		"qa":    "#ffd700",
		"dev":   "#663399",
		"load":  "#1e90ff",
		"beta":  "#00ff00",
	}
)

type Slack struct {
	Client     *slack.Client
	Token      string
	ChannelId  string
	WebhookUrl string
	SlackOff   bool
}

func NewSlackClient(slackOff bool) Slack {
	return Slack{
		Client:     slack.New(os.Getenv(SLACK_TOKEN)),
		Token:      os.Getenv(SLACK_TOKEN),
		WebhookUrl: os.Getenv(SLACK_WEBHOOK_URL),
		ChannelId:  os.Getenv(SLACK_CHANNEL),
		SlackOff:   slackOff,
	}
}

type SlackBody struct {
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Text  string `json:"text"`
	Color string `json:"color"`
}

func (s Slack) SendSimpleMessage(message string, env string) error {
	if !s.ValidClient() {
		return nil
	}
	if len(s.WebhookUrl) > 0 {
		if err := s.SendMessageWithWebhook(message, env); err != nil {
			return err
		}
	} else {
		color, ok := colorMapping[env]
		if !ok {
			color = "#ff0000"
		}
		attachment := slack.Attachment{
			Text:  message,
			Color: color,
		}
		msgOpt := slack.MsgOptionAttachments(attachment)

		if err := s.SendMessage(msgOpt); err != nil {
			return err
		}
	}

	return nil
}

func (s Slack) SendMessageWithWebhook(msg, env string) error {
	color, ok := colorMapping[env]
	if !ok {
		color = "#ff0000"
	}

	slackBody, _ := json.Marshal(SlackBody{
		Attachments: []Attachment{
			{
				Text:  msg,
				Color: color,
			},
		},
	})
	req, err := http.NewRequest(http.MethodPost, s.WebhookUrl, bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "ok" {
		return fmt.Errorf("Non-ok response returned from Slack")
	}
	return nil
}

func (s Slack) SendMessage(msgOpt slack.MsgOption) error {
	_, _, _, err := s.Client.SendMessage(s.ChannelId, msgOpt)
	if err != nil {
		return err
	}

	return nil
}

func (s Slack) CreateSimpleSection(text string) *slack.SectionBlock {
	txt := slack.NewTextBlockObject("mrkdwn", text, false, false)
	section := slack.NewSectionBlock(txt, nil, nil)
	return section
}

func (s Slack) CreateSimpleSectionWithFields(text string, fields []*slack.TextBlockObject) *slack.SectionBlock {
	txt := slack.NewTextBlockObject("mrkdwn", text, false, false)
	section := slack.NewSectionBlock(txt, fields, nil)
	fmt.Println(section.Fields[0].Text)
	return section
}

func (s Slack) CreateDividerSection() *slack.DividerBlock {
	return slack.NewDividerBlock()
}

func (s Slack) CreateSimpleAttachments(title, text string) slack.MsgOption {
	return slack.MsgOptionAttachments(
		slack.Attachment{
			Color:      "#36a64f",
			Title:      title,
			Text:       text,
			MarkdownIn: []string{"text"},
		},
	)
}

//ValidClient validates slack variables
func (s Slack) ValidClient() bool {
	if (len(s.WebhookUrl) == 0 && (len(s.Token) == 0 || len(s.ChannelId) == 0)) || s.SlackOff {
		return false
	}

	return true
}

func (s Slack) CreateTitleSection(text string) *slack.SectionBlock {
	txt := slack.NewTextBlockObject("mrkdwn", text, false, false)
	section := slack.NewSectionBlock(txt, nil, nil)
	return section
}
