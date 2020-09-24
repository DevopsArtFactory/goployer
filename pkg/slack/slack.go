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
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
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

// SendSimpleMessage creates and sends simple message
func (s Slack) SendSimpleMessage(message string) error {
	if !s.ValidClient() {
		return nil
	}
	if len(s.WebhookURL) > 0 {
		return s.SendMessageWithWebHook(message)
	} else {
		attachment := slack.Attachment{
			Text:  message,
			Color: constants.DefaultSlackColor,
		}
		msgOpt := slack.MsgOptionAttachments(attachment)

		if err := s.SendMessage(msgOpt); err != nil {
			return err
		}
	}

	return nil
}

// SendMessageWithWebhook is for WebhookURL
func (s Slack) SendMessageWithWebHook(msg string) error {
	slackBody, _ := json.Marshal(Body{
		Attachments: []Attachment{
			{
				Text:  msg,
				Color: constants.DefaultSlackColor,
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

func (s Slack) SendMessage(msgOpt ...slack.MsgOption) error {
	channel, timestamp, text, err := s.Client.SendMessage(s.ChannelID, msgOpt...)
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
			Color: constants.DefaultSlackColor,
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

// SendAPITestResultMessage sends API test message
func (s Slack) SendAPITestResultMessage(metrics []schemas.MetricResult) error {
	if len(s.WebhookURL) > 0 {
		return s.SendAPITestResultMessageWithWebHook(metrics)
	}

	var msgOpts []slack.MsgOption
	var attachments []slack.Attachment

	for _, m := range metrics {
		attachments = append(attachments, slack.Attachment{
			Color: constants.DefaultSlackColor,
			Text: fmt.Sprintf("*API*: %s\n*Duration*: %s\n*Wait*: %s\n*Requests*: %d\n*Rate*: %s\n*Throughput*: %s\n*Success*: %s\n*Latency P99*: %s\n",
				m.URL,
				tool.RoundTime(m.Data.Duration),
				tool.RoundTime(m.Data.Wait),
				m.Data.Requests,
				tool.RoundNum(m.Data.Rate),
				tool.RoundNum(m.Data.Throughput),
				tool.RoundNum(m.Data.Success),
				tool.RoundTime(m.Data.Latencies.P99),
			),
			MarkdownIn: []string{"text"},
		},
		)
	}

	msgOpts = append(msgOpts, slack.MsgOptionAttachments(attachments...))

	if err := s.SendMessage(msgOpts...); err != nil {
		return err
	}

	return nil
}

// SendAPITestResultMessageWithWebHook sends API test result with slack webhook
func (s Slack) SendAPITestResultMessageWithWebHook(metrics []schemas.MetricResult) error {
	attachments := []Attachment{}
	for _, m := range metrics {
		attachments = append(attachments, Attachment{
			Color: constants.DefaultSlackColor,
			Text: fmt.Sprintf("*API*: %s\n*Duration*: %s\n*Wait*: %s\n*Requests*: %d\n*Rate*: %s\n*Throughput*: %s\n*Success*: %s\n*Latency P99*: %s\n",
				m.URL,
				tool.RoundTime(m.Data.Duration),
				tool.RoundTime(m.Data.Wait),
				m.Data.Requests,
				tool.RoundNum(m.Data.Rate),
				tool.RoundNum(m.Data.Throughput),
				tool.RoundNum(m.Data.Success),
				tool.RoundTime(m.Data.Latencies.P99),
			),
		},
		)
	}

	slackBody, _ := json.Marshal(Body{
		Attachments: attachments,
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
