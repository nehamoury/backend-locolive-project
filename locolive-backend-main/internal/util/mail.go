package util

import (
	"fmt"
	"net/smtp"
)

type Mailer interface {
	SendResetEmail(toEmail string, token string) error
}

// Gmail/SMTP Implementation
type GmailMailer struct {
	senderName     string
	senderEmail    string
	senderPassword string // App Password
	host           string
	port           string
	frontendURL    string
}

func NewGmailMailer(name, email, password, host, port, frontendURL string) Mailer {
	return &GmailMailer{
		senderName:     name,
		senderEmail:    email,
		senderPassword: password,
		host:           host,
		port:           port,
		frontendURL:    frontendURL,
	}
}

func (m *GmailMailer) SendResetEmail(toEmail string, token string) error {
	if m.senderPassword == "" || m.senderPassword == "your_app_password" {
		fmt.Printf("------------\n[DEVELOPMENT MODE - EMAIL LOG]\nTo: %s\nSubject: Password Reset\nLink: %s/reset-password?token=%s\n------------\n", toEmail, m.frontendURL, token)
		return nil
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", m.frontendURL, token)
	subject := "Subject: Reset Your Locolive Password\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	
	body := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: auto; padding: 20px; border: 1px solid #eee; border-radius: 10px;">
			<div style="text-align: center; margin-bottom: 20px;">
				<h1 style="color: #FF3B8E; margin: 0; font-size: 28px;">Locolive</h1>
			</div>
			<div style="background: #f8fafc; padding: 30px; border-radius: 20px;">
				<h2 style="color: #1e293b; margin-top: 0;">Password Reset Request</h2>
				<p style="color: #475569; font-size: 16px; line-height: 1.6;">You requested a password reset for your Locolive account.</p>
				<p style="color: #475569; font-size: 16px; line-height: 1.6;">Click the button below to set a new password. This link expires in 15 minutes.</p>
				<div style="text-align: center; margin: 35px 0;">
					<a href="%s" style="background: linear-gradient(to right, #FF3B8E, #A855F7); color: white; padding: 14px 32px; text-decoration: none; border-radius: 14px; font-weight: bold; font-size: 16px; display: inline-block; box-shadow: 0 10px 15px -3px rgba(255, 59, 142, 0.3);">Reset Password</a>
				</div>
				<p style="color: #94a3b8; font-size: 13px; text-align: center;">If you didn't request this, you can safely ignore this email.</p>
			</div>
			<p style="color: #cbd5e1; font-size: 11px; text-align: center; margin-top: 20px;">&copy; 2026 Locolive. All rights reserved.</p>
		</div>
	`, resetLink)

	msg := []byte(subject + mime + body)
	auth := smtp.PlainAuth("", m.senderEmail, m.senderPassword, m.host)

	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	err := smtp.SendMail(addr, auth, m.senderEmail, []string{toEmail}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	return nil
}


