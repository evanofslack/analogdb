package logger

import (
	"fmt"
	"sync"
	"time"

	"github.com/ashwanthkumar/slack-go-webhook"
	"github.com/rs/zerolog"
)

const notifyInterval = 5 * time.Minute

var wg sync.WaitGroup

type notification struct {
	message              string
	timeSinceLastNotify  time.Time
	countSinceLastNotify int
}

func newNotification(message string) notification {
	return notification{
		message:              message,
		timeSinceLastNotify:  time.Now(),
		countSinceLastNotify: 0,
	}
}

type SlackHook struct {
	url           string
	notifications map[string]notification
}

func newSlackNotifier(url string) *SlackHook {
	return &SlackHook{
		url:           url,
		notifications: make(map[string]notification),
	}
}

func (slackhook *SlackHook) shouldNotify(message string) bool {

	var notification notification
	var found bool

	// Have we already seen this message?
	if notification, found = slackhook.notifications[message]; !found {

		// no we have not, track it and send notification
		slackhook.notifications[message] = newNotification(message)
		return true
	}

	// yes we have, increase the count
	notification.countSinceLastNotify++

	// only send another notification if we are past notify interval
	if time.Since(notification.timeSinceLastNotify) > notifyInterval {
		notification.timeSinceLastNotify = time.Now()
		notification.countSinceLastNotify = 0
		return true
	}

	// don't notify
	return false
}

func (slackhook *SlackHook) Run(
	e *zerolog.Event,
	level zerolog.Level,
	message string,
) {
	// if the level is less than our notify threshold, don't notify
	if level <= zerolog.WarnLevel {
		return
	}

	// if we have already sent this same notification
	// within notify interval, don't notify
	if !slackhook.shouldNotify(message) {
		return
	}

	// send notification to slack
	wg.Add(1)
	go func() {
		slackhook.notify(message)
		wg.Done()
	}()
}

func (slackhook *SlackHook) notify(message string) {

	analogdbLink := slack.Attachment{}
	analogdbLink.AddAction(slack.Action{Type: "button", Text: "check analogdb.com", Url: "https://analogdb.com", Style: "primary"})
	payload := slack.Payload{
		Text:        message,
		IconEmoji:   ":golang:",
		Attachments: []slack.Attachment{analogdbLink},
	}
	err := slack.Send(slackhook.url, "", payload)
	if len(err) > 0 {
		fmt.Printf("Failed to send error message to slack webhook: %s", err)
	}

}
