package notification

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type NotificationService struct {
	app *firebase.App
}

func NewNotificationService(credentialsPath string) (*NotificationService, error) {
	opt := option.WithCredentialsFile(credentialsPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %v", err)
	}

	return &NotificationService{
		app: app,
	}, nil
}

// SendPushNotification sends a notification to a single device token.
// Returns a boolean indicating if the token is invalid/expired and should be removed.
func (s *NotificationService) SendPushNotification(ctx context.Context, token string, title, body string, data map[string]string) (bool, error) {
	client, err := s.app.Messaging(ctx)
	if err != nil {
		return false, err
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:  data,
		Token: token,
		Webpush: &messaging.WebpushConfig{
			Notification: &messaging.WebpushNotification{
				Icon: "/pwa-192x192.png",
			},
			FCMOptions: &messaging.WebpushFCMOptions{
				Link: "/notifications",
			},
		},
	}

	response, err := client.Send(ctx, message)
	if err != nil {
		// Check if the error is due to an invalid or unregistered token
		if messaging.IsUnregistered(err) || messaging.IsInvalidArgument(err) {
			log.Printf("Token %s is invalid or unregistered, should be removed: %v", token, err)
			return true, nil
		}
		return false, err
	}

	log.Printf("Successfully sent message: %s", response)
	return false, nil
}

func (s *NotificationService) SendMulticastNotification(ctx context.Context, tokens []string, title, body string, data map[string]string) ([]string, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	client, err := s.app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:   data,
		Tokens: tokens,
		Webpush: &messaging.WebpushConfig{
			Notification: &messaging.WebpushNotification{
				Icon: "/pwa-192x192.png",
			},
		},
	}

	br, err := client.SendMulticast(ctx, message)
	if err != nil {
		return nil, err
	}

	var invalidTokens []string
	if br.FailureCount > 0 {
		for idx, resp := range br.Responses {
			if !resp.Success {
				if messaging.IsUnregistered(resp.Error) {
					invalidTokens = append(invalidTokens, tokens[idx])
				}
			}
		}
	}

	log.Printf("%d messages were sent successfully, %d failed", br.SuccessCount, br.FailureCount)
	return invalidTokens, nil
}

