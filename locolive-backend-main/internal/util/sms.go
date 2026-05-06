package util

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type SMSProvider interface {
	SendOTP(toPhone string, code string) error
}

type TwilioProvider struct {
	accountSid string
	authToken  string
	fromNumber string
}

func NewTwilioProvider(sid, token, from string) SMSProvider {
	if sid == "" || sid == "your_twilio_sid" {
		// In production, NEVER allow dev bypass
		env := os.Getenv("ENVIRONMENT")
		if env == "production" || env == "prod" {
			panic("CRITICAL: Twilio credentials not configured in production. Set TWILIO_ACCOUNT_SID in app.env")
		}
		return &DevSMSProvider{}
	}
	return &TwilioProvider{
		accountSid: sid,
		authToken:  token,
		fromNumber: from,
	}
}

func (t *TwilioProvider) SendOTP(toPhone string, code string) error {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSid)

	msgData := url.Values{}
	msgData.Set("To", toPhone)
	msgData.Set("From", t.fromNumber)
	msgData.Set("Body", fmt.Sprintf("Your Locolive verification code is: %s. Valid for 10 minutes.", code))

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(msgData.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(t.accountSid, t.authToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("twilio API returned error status: %s", resp.Status)
	}

	return nil
}

type DevSMSProvider struct{}

func (d *DevSMSProvider) SendOTP(toPhone string, code string) error {
	fmt.Printf("------------\n[DEVELOPMENT MODE - SMS LOG]\nTo: %s\nMessage: Your Locolive verification code is: %s\n------------\n", toPhone, code)
	return nil
}
