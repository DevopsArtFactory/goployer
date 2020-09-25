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
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"

	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/DevopsArtFactory/goployer/pkg/constants"
	"github.com/DevopsArtFactory/goployer/pkg/schemas"
	"github.com/DevopsArtFactory/goployer/pkg/tool"
)

type Slack struct {
	Client     *slack.Client
	Token      string
	ChannelID  string
	WebhookURL string
	SlackOff   bool
	Color      string
}

// NewSlackClient creates new slack client
func NewSlackClient(slackOff bool) Slack {
	return Slack{
		Client:     slack.New(os.Getenv(constants.SlackToken)),
		Token:      os.Getenv(constants.SlackToken),
		WebhookURL: os.Getenv(constants.SlackWebHookURL),
		ChannelID:  os.Getenv(constants.SlackChannel),
		SlackOff:   slackOff,
		Color:      tool.GetRandomRGBColor(),
	}
}

type Body struct {
	Blocks      []Block      `json:"blocks,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Block struct {
	Type string `json:"type"`
	Text *Text  `json:"text,omitempty"`
}

type Text struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

type Attachment struct {
	Text   string  `json:"text"`
	Color  string  `json:"color"`
	Fields []Field `json:"fields"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SendSimpleMessage creates and sends simple message
func (s Slack) SendSimpleMessage(message string) error {
	if !s.ValidClient() {
		return nil
	}
	if len(s.WebhookURL) > 0 {
		return s.SendMessageWithWebHook(message)
	}
	attachment := slack.Attachment{
		Text:  message,
		Color: s.Color,
	}
	msgOpt := slack.MsgOptionAttachments(attachment)

	if err := s.SendMessage(msgOpt); err != nil {
		return err
	}

	return nil
}

// SendMessageWithWebhook is for WebhookURL
func (s Slack) SendMessageWithWebHook(msg string) error {
	slackBody, _ := json.Marshal(Body{
		Attachments: []Attachment{
			{
				Text:  msg,
				Color: s.Color,
			},
		},
	})

	return sendSlackRequest(slackBody, s.WebhookURL)
}

// SendMessage really sends message with token
func (s Slack) SendMessage(msgOpt ...slack.MsgOption) error {
	channel, timestamp, text, err := s.Client.SendMessage(s.ChannelID, msgOpt...)
	if err != nil {
		return err
	}

	logrus.Debugf("channel: %s, timestamp: %s, text: %s", channel, timestamp, text)
	return nil
}

// CreateSimpleSection creates simple section with text
func (s Slack) CreateSimpleSection(text string) *slack.SectionBlock {
	txt := slack.NewTextBlockObject("mrkdwn", text, false, false)
	section := slack.NewSectionBlock(txt, nil, nil)
	return section
}

// CreateDividerSection creates a new division block
func (s Slack) CreateDividerSection() *slack.DividerBlock {
	return slack.NewDividerBlock()
}

//ValidClient validates slack variables
func (s Slack) ValidClient() bool {
	if (len(s.WebhookURL) == 0 && (len(s.Token) == 0 || len(s.ChannelID) == 0)) || s.SlackOff {
		return false
	}

	return true
}

// CreateTitleSection creates title section
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
			Color: s.Color,
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
	var attachments []Attachment
	for _, m := range metrics {
		attachments = append(attachments, Attachment{
			Color: s.Color,
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

	return sendSlackRequest(slackBody, s.WebhookURL)
}

// SendSummaryMessage sends summary of deployment
func (s Slack) SendSummaryMessage(config schemas.Config, stacks []schemas.Stack, app string) error {
	if len(s.WebhookURL) > 0 {
		return s.SendSummaryMessageWithWebHook(config, stacks, app)
	}

	var msgOpts []slack.MsgOption
	var attachments []slack.Attachment

	titleSection := s.CreateTitleSection(fmt.Sprintf("*[ %s ] Deployment has been started*", app))

	msgOpts = append(msgOpts, slack.MsgOptionBlocks(
		titleSection,
		s.CreateDividerSection(),
	))

	// configurations
	data := builder.ExtractAppliedConfig(config)
	var fields []slack.AttachmentField
	for _, d := range data {
		fields = append(fields, slack.AttachmentField{
			Title: d[0],
			Value: d[1],
			Short: true,
		})
	}
	attachments = append(attachments, slack.Attachment{
		Color:  s.Color,
		Text:   "`These are configurations that are applied to this deployment.`",
		Fields: fields,
	})

	for _, st := range stacks {
		var fs []slack.AttachmentField
		val := reflect.ValueOf(&st).Elem()

		for i := 0; i < val.NumField(); i++ {
			typeField := val.Type().Field(i)
			key := strings.ReplaceAll(typeField.Tag.Get("yaml"), "_", "-")
			key = strings.ReplaceAll(key, ",omitempty", "")
			t := val.FieldByName(typeField.Name)
			switch t.Kind() {
			case reflect.String:
				if len(val.FieldByName(typeField.Name).String()) > 0 && typeField.Name != "Stack" {
					fs = append(fs, slack.AttachmentField{
						Title: key,
						Value: val.FieldByName(typeField.Name).String(),
						Short: true,
					})
				}
			case reflect.Int, reflect.Int64:
				if val.FieldByName(typeField.Name).Int() > 0 {
					var value string
					if tool.IsStringInArray(key, []string{"polling-interval", "timeout"}) {
						value = fmt.Sprintf("%.0fs", time.Duration(val.FieldByName(typeField.Name).Int()).Seconds())
					} else {
						value = fmt.Sprintf("%d", val.FieldByName(typeField.Name).Int())
					}
					fs = append(fs, slack.AttachmentField{
						Title: key,
						Value: value,
						Short: true,
					})
				}
			case reflect.Bool:
				fs = append(fs, slack.AttachmentField{
					Title: key,
					Value: fmt.Sprintf("%t", val.FieldByName(typeField.Name).Bool()),
					Short: true,
				})
			default:
				switch key {
				case "capacity":
					fs = append(fs, slack.AttachmentField{
						Title: key,
						Value: fmt.Sprintf("min: %d, desired: %d, max: %d", st.Capacity.Min, st.Capacity.Desired, st.Capacity.Max),
						Short: true,
					})
				case "tags":
					fs = append(fs, slack.AttachmentField{
						Title: key,
						Value: strings.Join(st.Tags, ","),
						Short: true,
					})
				case "block-devices":
					if len(st.BlockDevices) > 0 {
						var bstr string
						for _, bd := range st.BlockDevices {
							bstr += fmt.Sprintf("%s | %s | %d | %d\n", bd.DeviceName, bd.VolumeType, bd.VolumeSize, bd.Iops)
						}
						fs = append(fs, slack.AttachmentField{
							Title: fmt.Sprintf("%s(name|type|size|iops)", key),
							Value: bstr,
							Short: true,
						})
					}
				case "regions":
					for _, region := range st.Regions {
						if len(config.Region) == 0 || config.Region == region.Region {
							rv := reflect.ValueOf(&region).Elem()
							for i := 0; i < rv.NumField(); i++ {
								rtf := rv.Type().Field(i)
								rk := strings.ReplaceAll(rtf.Tag.Get("yaml"), "_", "-")
								rk = strings.ReplaceAll(rk, ",omitempty", "")
								rt := rv.FieldByName(rtf.Name)
								switch rt.Kind() {
								case reflect.String:
									if len(rv.FieldByName(rtf.Name).String()) > 0 && rtf.Name != "Region" {
										fs = append(fs, slack.AttachmentField{
											Title: fmt.Sprintf("[%s] %s", region.Region, rk),
											Value: rv.FieldByName(rtf.Name).String(),
											Short: true,
										})
									}
								case reflect.Int, reflect.Int64:
									if val.FieldByName(rtf.Name).Int() > 0 {
										fs = append(fs, slack.AttachmentField{
											Title: fmt.Sprintf("[%s] %s", region.Region, rk),
											Value: fmt.Sprintf("%d", rv.FieldByName(rtf.Name).Int()),
											Short: true,
										})
									}
								case reflect.Bool:
									fs = append(fs, slack.AttachmentField{
										Title: fmt.Sprintf("[%s] %s", region.Region, rk),
										Value: fmt.Sprintf("%t", rv.FieldByName(rtf.Name).Bool()),
										Short: true,
									})
								case reflect.Slice:
									if rt.Len() > 0 {
										var value string
										if rt.Index(0).Kind() == reflect.String {
											for j := 0; j < rt.Len(); j++ {
												value += fmt.Sprintf("%s\n", rt.Index(j).String())
											}
											fs = append(fs, slack.AttachmentField{
												Title: fmt.Sprintf("[%s] %s", region.Region, rk),
												Value: value,
												Short: true,
											})
										}
									}
								}
							}
						}
					}
				}
			}
		}
		attachments = append(attachments, slack.Attachment{
			Color:  s.Color,
			Text:   fmt.Sprintf("`Stack configurations: %s`", st.Stack),
			Fields: fs,
		})
	}

	msgOpts = append(msgOpts, slack.MsgOptionAttachments(attachments...))

	if err := s.SendMessage(msgOpts...); err != nil {
		return err
	}
	return nil
}

// SendSummaryMessageWithWebHook sends summary message
func (s Slack) SendSummaryMessageWithWebHook(config schemas.Config, stacks []schemas.Stack, app string) error {
	var attachments []Attachment
	var blocks []Block

	// title
	blocks = append(blocks, Block{
		Type: "section",
		Text: &Text{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*[ %s ] Deployment has been started*", app),
		},
	})

	// divider
	blocks = append(blocks, Block{
		Type: "divider",
	})

	// configurations
	data := builder.ExtractAppliedConfig(config)
	var fields []Field
	for _, d := range data {
		fields = append(fields, Field{
			Title: d[0],
			Value: d[1],
			Short: true,
		})
	}
	attachments = append(attachments, Attachment{
		Color:  s.Color,
		Text:   "`These are configurations that are applied to this deployment.`",
		Fields: fields,
	})

	for _, st := range stacks {
		var fs []Field
		val := reflect.ValueOf(&st).Elem()

		for i := 0; i < val.NumField(); i++ {
			typeField := val.Type().Field(i)
			key := strings.ReplaceAll(typeField.Tag.Get("yaml"), "_", "-")
			key = strings.ReplaceAll(key, ",omitempty", "")
			t := val.FieldByName(typeField.Name)
			switch t.Kind() {
			case reflect.String:
				if len(val.FieldByName(typeField.Name).String()) > 0 && typeField.Name != "Stack" {
					fs = append(fs, Field{
						Title: key,
						Value: val.FieldByName(typeField.Name).String(),
						Short: true,
					})
				}
			case reflect.Int, reflect.Int64:
				if val.FieldByName(typeField.Name).Int() > 0 {
					var value string
					if tool.IsStringInArray(key, []string{"polling-interval", "timeout"}) {
						value = fmt.Sprintf("%.0fs", time.Duration(val.FieldByName(typeField.Name).Int()).Seconds())
					} else {
						value = fmt.Sprintf("%d", val.FieldByName(typeField.Name).Int())
					}
					fs = append(fs, Field{
						Title: key,
						Value: value,
						Short: true,
					})
				}
			case reflect.Bool:
				fs = append(fs, Field{
					Title: key,
					Value: fmt.Sprintf("%t", val.FieldByName(typeField.Name).Bool()),
					Short: true,
				})
			default:
				switch key {
				case "capacity":
					fs = append(fs, Field{
						Title: key,
						Value: fmt.Sprintf("min: %d, desired: %d, max: %d", st.Capacity.Min, st.Capacity.Desired, st.Capacity.Max),
						Short: true,
					})
				case "tags":
					fs = append(fs, Field{
						Title: key,
						Value: strings.Join(st.Tags, ","),
						Short: true,
					})
				case "block-devices":
					if len(st.BlockDevices) > 0 {
						var bstr string
						for _, bd := range st.BlockDevices {
							bstr += fmt.Sprintf("%s | %s | %d | %d\n", bd.DeviceName, bd.VolumeType, bd.VolumeSize, bd.Iops)
						}
						fs = append(fs, Field{
							Title: fmt.Sprintf("%s(name|type|size|iops)", key),
							Value: bstr,
							Short: true,
						})
					}
				case "regions":
					for _, region := range st.Regions {
						if len(config.Region) == 0 || config.Region == region.Region {
							rv := reflect.ValueOf(&region).Elem()
							for i := 0; i < rv.NumField(); i++ {
								rtf := rv.Type().Field(i)
								rk := strings.ReplaceAll(rtf.Tag.Get("yaml"), "_", "-")
								rk = strings.ReplaceAll(rk, ",omitempty", "")
								rt := rv.FieldByName(rtf.Name)
								switch rt.Kind() {
								case reflect.String:
									if len(rv.FieldByName(rtf.Name).String()) > 0 && rtf.Name != "Region" {
										fs = append(fs, Field{
											Title: fmt.Sprintf("[%s] %s", region.Region, rk),
											Value: rv.FieldByName(rtf.Name).String(),
											Short: true,
										})
									}
								case reflect.Int, reflect.Int64:
									if val.FieldByName(rtf.Name).Int() > 0 {
										fs = append(fs, Field{
											Title: fmt.Sprintf("[%s] %s", region.Region, rk),
											Value: fmt.Sprintf("%d", rv.FieldByName(rtf.Name).Int()),
											Short: true,
										})
									}
								case reflect.Bool:
									fs = append(fs, Field{
										Title: fmt.Sprintf("[%s] %s", region.Region, rk),
										Value: fmt.Sprintf("%t", rv.FieldByName(rtf.Name).Bool()),
										Short: true,
									})
								case reflect.Slice:
									if rt.Len() > 0 {
										var value string
										if rt.Index(0).Kind() == reflect.String {
											for j := 0; j < rt.Len(); j++ {
												value += fmt.Sprintf("%s\n", rt.Index(j).String())
											}
											fs = append(fs, Field{
												Title: fmt.Sprintf("[%s] %s", region.Region, rk),
												Value: value,
												Short: true,
											})
										}
									}
								}
							}
						}
					}
				}
			}
		}
		attachments = append(attachments, Attachment{
			Color:  s.Color,
			Text:   fmt.Sprintf("`Stack configurations: %s`", st.Stack),
			Fields: fs,
		})
	}

	slackBody, _ := json.Marshal(Body{
		Attachments: attachments,
		Blocks:      blocks,
	})

	return sendSlackRequest(slackBody, s.WebhookURL)
}

// sendSlackRequest sends request for slack message
func sendSlackRequest(slackBody []byte, url string) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(slackBody))
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
