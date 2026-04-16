package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Mailer interface {
	SendResetEmail(toEmail string, token string) error
}

type SendGridMailer struct {
	apiKey      string
	fromEmail   string
	frontendURL string
}

func NewSendGridMailer(apiKey, fromEmail, frontendURL string) Mailer {
	return &SendGridMailer{
		apiKey:      apiKey,
		fromEmail:   fromEmail,
		frontendURL: frontendURL,
	}
}

type sendGridRequest struct {
	Personalizations []personalization `json:"personalizations"`
	From             emailUser         `json:"from"`
	Subject          string            `json:"subject"`
	Content          []mailContent     `json:"content"`
}

type personalization struct {
	To []emailUser `json:"to"`
}

type emailUser struct {
	Email string `json:"email"`
}

type mailContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (m *SendGridMailer) SendResetEmail(toEmail string, token string) error {
	if m.apiKey == "" || m.apiKey == "your_sendgrid_api_key" {
		fmt.Printf("------------\n[DEVELOPMENT MODE - EMAIL LOG]\nTo: %s\nSubject: Password Reset\nLink: %s/reset-password?token=%s\n------------\n", toEmail, m.frontendURL, token)
		return nil
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", m.frontendURL, token)
	
	reqBody := sendGridRequest{
		Personalizations: []personalization{
			{
				To: []emailUser{{Email: toEmail}},
			},
		},
		From:    emailUser{Email: m.fromEmail},
		Subject: "Reset Your Locolive Password",
		Content: []mailContent{
			{
				Type: "text/html",
				Value: fmt.Sprintf(`
					<h1>Password Reset Request</h1>
					<p>You requested a password reset for your Locolive account.</p>
					<p>Click the link below to set a new password. This link expires in 15 minutes.</p>
					<a href="%s" style="background-color: #4F46E5; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Reset Password</a>
					<p>If you didn't request this, you can safely ignore this email.</p>
				`, resetLink),
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid error: status code %d", resp.StatusCode)
	}

	return nil
}
