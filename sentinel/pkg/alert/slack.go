package alert

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bisonai.com/orakl/sentinel/pkg/secrets"
	"github.com/rs/zerolog/log"
)

var WEBHOOK_ENDPOINT string

func init() {
	// TODO load inside of env.go
	WEBHOOK_ENDPOINT = loadWebhookEndpoint()
}

func loadWebhookEndpoint() string {
	endpoint := secrets.GetSecret("SLACK_WEBHOOK_ENDPOINT")
	if endpoint == "" && flag.Lookup("test.v") != nil {
		log.Error().Msg("SLACK_WEBHOOK_ENDPOINT not found")
	}
	return endpoint
}

func SlackAlert(text string) {
	payload := map[string]string{
		"text": text,
	}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(WEBHOOK_ENDPOINT, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("Slack Alert: Error sending message: %s", err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Error().Msg(fmt.Sprintf("Slack Alert: Non-OK HTTP status: %s", resp.Status))
		return
	}
	log.Debug().Msg(fmt.Sprintf("Slack Response Status: %s", resp.Status))
}

const DefaultSlackTimeout = 5 * time.Second

type SlackClient struct {
	WebHookUrl string
	UserName   string
	Channel    string
	TimeOut    time.Duration
}

type SimpleSlackRequest struct {
	Text      string
	IconEmoji string
}

type SlackJobNotification struct {
	Color     string
	IconEmoji string
	Details   string
	Text      string
}

type SlackMessage struct {
	Username    string       `json:"username,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Channel     string       `json:"channel,omitempty"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Color         string      `json:"color,omitempty"`
	Fallback      string      `json:"fallback,omitempty"`
	CallbackID    string      `json:"callback_id,omitempty"`
	ID            int         `json:"id,omitempty"`
	AuthorID      string      `json:"author_id,omitempty"`
	AuthorName    string      `json:"author_name,omitempty"`
	AuthorSubname string      `json:"author_subname,omitempty"`
	AuthorLink    string      `json:"author_link,omitempty"`
	AuthorIcon    string      `json:"author_icon,omitempty"`
	Title         string      `json:"title,omitempty"`
	TitleLink     string      `json:"title_link,omitempty"`
	Pretext       string      `json:"pretext,omitempty"`
	Text          string      `json:"text,omitempty"`
	ImageURL      string      `json:"image_url,omitempty"`
	ThumbURL      string      `json:"thumb_url,omitempty"`
	MarkdownIn    []string    `json:"mrkdwn_in,omitempty"`
	Ts            json.Number `json:"ts,omitempty"`
}

func (sc SlackClient) SendSlackNotification(sr SimpleSlackRequest) error {
	slackRequest := SlackMessage{
		Text:      sr.Text,
		Username:  sc.UserName,
		IconEmoji: sr.IconEmoji,
		Channel:   sc.Channel,
	}
	return sc.sendHttpRequest(slackRequest)
}

func (sc SlackClient) SendJobNotification(job SlackJobNotification) error {
	attachment := Attachment{
		Color: job.Color,
		Text:  job.Details,
		Ts:    json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
	}
	slackRequest := SlackMessage{
		Text:        job.Text,
		Username:    sc.UserName,
		IconEmoji:   job.IconEmoji,
		Channel:     sc.Channel,
		Attachments: []Attachment{attachment},
	}
	return sc.sendHttpRequest(slackRequest)
}

func (sc SlackClient) SendError(message string, options ...string) (err error) {
	return sc.funcName("danger", message, options)
}

func (sc SlackClient) SendInfo(message string, options ...string) (err error) {
	return sc.funcName("good", message, options)
}

func (sc SlackClient) SendWarning(message string, options ...string) (err error) {
	return sc.funcName("warning", message, options)
}

func (sc SlackClient) funcName(color string, message string, options []string) error {
	emoji := ":hammer_and_wrench"
	if len(options) > 0 {
		emoji = options[0]
	}
	sjn := SlackJobNotification{
		Color:     color,
		IconEmoji: emoji,
		Details:   message,
	}
	return sc.SendJobNotification(sjn)
}
func (sc SlackClient) sendHttpRequest(slackRequest SlackMessage) error {
	slackBody, _ := json.Marshal(slackRequest)
	req, err := http.NewRequest(http.MethodPost, sc.WebHookUrl, bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	if sc.TimeOut == 0 {
		sc.TimeOut = DefaultSlackTimeout
	}
	client := &http.Client{Timeout: sc.TimeOut}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-ok response returned from slack: %s", resp.Status)
	}
	return nil
}
