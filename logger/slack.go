package logger

import (
    "sync"
    "fmt"

    "github.com/ashwanthkumar/slack-go-webhook"
    "github.com/rs/zerolog"
)

var wg sync.WaitGroup

type SlackHook struct{
	url string
}

func newSlackNotifier(url string) *SlackHook {
	return &SlackHook{url: url}
}

func (slackhook *SlackHook) Run(
    e *zerolog.Event,
    level zerolog.Level,
    message string,
) {
    if level > zerolog.WarnLevel {
        wg.Add(1)
        go func() {
            slackhook.notify(message)
            wg.Done()
        }()
    }
}

func (slackhook *SlackHook) notify(message string) {

    analogdbLink := slack.Attachment {}
    analogdbLink.AddAction(slack.Action { Type: "button", Text: "check analogdb.com", Url: "https://analogdb.com", Style: "primary" })
    payload := slack.Payload {
      Text: message,
      IconEmoji: ":golang:",
      Attachments: []slack.Attachment{analogdbLink},
    }
    err := slack.Send(slackhook.url, "", payload)
    if len(err) > 0 {
      fmt.Printf("Failed to send error message to slack webhook: %s", err)
    }

}


