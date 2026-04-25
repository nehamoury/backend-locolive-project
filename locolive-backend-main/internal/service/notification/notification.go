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

func (s *NotificationService) SendPushNotification(ctx context.Context, token string, title, body string, data map[string]string) error {
	client, err := s.app.Messaging(ctx)
	if err != nil {
		return err
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:  data,
		Token: token,
	}

	response, err := client.Send(ctx, message)
	if err != nil {
		return err
	}

	log.Printf("Successfully sent message: %s", response)
	return nil
}

func (s *NotificationService) SendMulticastNotification(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	if len(tokens) == 0 {
		return nil
	}

	client, err := s.app.Messaging(ctx)
	if err != nil {
		return err
	}

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:   data,
		Tokens: tokens,
	}

	br, err := client.SendMulticast(ctx, message)
	if err != nil {
		return err
	}

	log.Printf("%d messages were sent successfully", br.SuccessCount)
	return nil
}
