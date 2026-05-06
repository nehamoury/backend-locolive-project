package util

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

var firebaseAuthClient *auth.Client

// InitFirebaseAuth initializes the Firebase Auth client using service account credentials
func InitFirebaseAuth(credentialsPath string) error {
	opt := option.WithCredentialsFile(credentialsPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return fmt.Errorf("error initializing firebase app: %v", err)
	}

	client, err := app.Auth(context.Background())
	if err != nil {
		return fmt.Errorf("error getting firebase auth client: %v", err)
	}

	firebaseAuthClient = client
	log.Println("Firebase Auth initialized successfully")
	return nil
}

// VerifyFirebasePhoneToken verifies a Firebase ID token and returns the phone number
func VerifyFirebasePhoneToken(idToken string) (string, error) {
	if firebaseAuthClient == nil {
		return "", fmt.Errorf("firebase auth client not initialized")
	}

	token, err := firebaseAuthClient.VerifyIDToken(context.Background(), idToken)
	if err != nil {
		return "", fmt.Errorf("invalid firebase token: %v", err)
	}

	// Check if token contains phone number (from phone auth)
	phone, ok := token.Claims["phone_number"].(string)
	if !ok || phone == "" {
		return "", fmt.Errorf("token does not contain a verified phone number")
	}

	return phone, nil
}
