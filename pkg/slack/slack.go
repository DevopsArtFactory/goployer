/*
copyright 2020 the Goployer authors

licensed under the apache license, version 2.0 (the "license");
you may not use this file except in compliance with the license.
you may obtain a copy of the license at

    http://www.apache.org/licenses/license-2.0

unless required by applicable law or agreed to in writing, software
distributed under the license is distributed on an "as is" basis,
without warranties or conditions of any kind, either express or implied.
see the license for the specific language governing permissions and
limitations under the license.
*/

package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

var (
	slackToken      = "SLACK_TOKEN"
	slackChannel    = "SLACK_CHANNEL"
	slackWebhookURL = "SLACK_WEBHOOK_URL"

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
	ChannelID  string
	WebhookURL string
	SlackOff   bool
}

func NewSlackClient(slackOff bool) Slack {
	return Slack{
		Client:     slack.New(os.Getenv(slackToken)),
		Token:      os.Getenv(slackToken),
		WebhookURL: os.Getenv(slackWebhookURL),
		ChannelID:  os.Getenv(slackChannel),
		SlackOff:   slackOff,
	}
}

type Body struct {
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
	if len(s.WebhookURL) > 0 {
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

	slackBody, _ := json.Marshal(Body{
		Attachments: []Attachment{
			{
				Text:  msg,
				Color: color,
			},
		},
	})
	req, err := http.NewRequest(http.MethodPost, s.WebhookURL, bytes.NewBuffer(slackBody))
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
		return errors.New("non-ok response returned from Slack")
	}

	resp.Body.Close()

	return nil
}

func (s Slack) SendMessage(msgOpt slack.MsgOption) error {
	channel, timestamp, text, err := s.Client.SendMessage(s.ChannelID, msgOpt)
	if err != nil {
		return err
	}

	logrus.Debugf("channel: %s, timestamp: %s, text: %s", channel, timestamp, text)

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
	if (len(s.WebhookURL) == 0 && (len(s.Token) == 0 || len(s.ChannelID) == 0)) || s.SlackOff {
		return false
	}

	return true
}

func (s Slack) CreateTitleSection(text string) *slack.SectionBlock {
	txt := slack.NewTextBlockObject("mrkdwn", text, false, false)
	section := slack.NewSectionBlock(txt, nil, nil)
	return section
}
