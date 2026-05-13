package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
)

type Mailer interface {
	SendResetEmail(toEmail string, token string) error
	SendVerificationEmail(toEmail string, token string) error
	SendOTPEmail(toEmail string, otp string) error
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
		fmt.Printf("[EMAIL ERROR] Failed to send email to %s: %v\n", toEmail, err)
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	fmt.Printf("[EMAIL SUCCESS] Sent reset email to %s\n", toEmail)
	return nil
}

func (m *GmailMailer) SendVerificationEmail(toEmail string, token string) error {
	if m.senderPassword == "" || m.senderPassword == "your_app_password" {
		fmt.Printf("------------\n[DEVELOPMENT MODE - EMAIL LOG]\nTo: %s\nSubject: Verify Your Email\nLink: %s/verify-email?token=%s\n------------\n", toEmail, m.frontendURL, token)
		return nil
	}

	verificationLink := fmt.Sprintf("%s/verify-email?token=%s", m.frontendURL, token)
	subject := "Subject: Verify Your Locolive Email\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	body := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: auto; padding: 20px; border: 1px solid #eee; border-radius: 10px;">
			<div style="text-align: center; margin-bottom: 20px;">
				<h1 style="color: #FF3B8E; margin: 0; font-size: 28px;">Locolive</h1>
			</div>
			<div style="background: #f8fafc; padding: 30px; border-radius: 20px;">
				<h2 style="color: #1e293b; margin-top: 0;">Verify Your Email Address</h2>
				<p style="color: #475569; font-size: 16px; line-height: 1.6;">Welcome to Locolive! Please verify your email address to activate your account.</p>
				<p style="color: #475569; font-size: 16px; line-height: 1.6;">Click the button below to verify. This link expires in 24 hours.</p>
				<div style="text-align: center; margin: 35px 0;">
					<a href="%s" style="background: linear-gradient(to right, #FF3B8E, #A855F7); color: white; padding: 14px 32px; text-decoration: none; border-radius: 14px; font-weight: bold; font-size: 16px; display: inline-block; box-shadow: 0 10px 15px -3px rgba(255, 59, 142, 0.3);">Verify Email</a>
				</div>
				<p style="color: #94a3b8; font-size: 13px; text-align: center;">If you didn't create an account, you can safely ignore this email.</p>
			</div>
			<p style="color: #cbd5e1; font-size: 11px; text-align: center; margin-top: 20px;">&copy; 2026 Locolive. All rights reserved.</p>
		</div>
	`, verificationLink)

	msg := []byte(subject + mime + body)
	auth := smtp.PlainAuth("", m.senderEmail, m.senderPassword, m.host)

	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	err := smtp.SendMail(addr, auth, m.senderEmail, []string{toEmail}, msg)
	if err != nil {
		fmt.Printf("[EMAIL ERROR] Failed to send verification email to %s: %v\n", toEmail, err)
		return fmt.Errorf("failed to send verification email via SMTP: %w", err)
	}

	fmt.Printf("[EMAIL SUCCESS] Sent verification email to %s\n", toEmail)
	return nil
}

func (m *GmailMailer) SendOTPEmail(toEmail string, otp string) error {
	if m.senderPassword == "" || m.senderPassword == "your_app_password" {
		fmt.Printf("------------\n[DEVELOPMENT MODE - EMAIL LOG]\nTo: %s\nSubject: Your Locolive Verification Code\nOTP: %s\n------------\n", toEmail, otp)
		return nil
	}

	subject := "Subject: Your Locolive Verification Code\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	body := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: auto; padding: 20px; border: 1px solid #eee; border-radius: 10px;">
			<div style="text-align: center; margin-bottom: 20px;">
				<h1 style="color: #FF3B8E; margin: 0; font-size: 28px;">Locolive</h1>
			</div>
			<div style="background: #f8fafc; padding: 30px; border-radius: 20px;">
				<h2 style="color: #1e293b; margin-top: 0;">Verify Your Account</h2>
				<p style="color: #475569; font-size: 16px; line-height: 1.6;">Welcome to Locolive! Use the code below to verify your account.</p>
				<div style="text-align: center; margin: 35px 0;">
					<div style="background: #1e293b; color: white; padding: 20px 40px; border-radius: 14px; font-size: 42px; font-weight: 900; letter-spacing: 12px; display: inline-block; font-family: monospace;">%s</div>
				</div>
				<p style="color: #475569; font-size: 16px; line-height: 1.6;">Enter this code on the verification page. It expires in 24 hours.</p>
				<p style="color: #94a3b8; font-size: 13px; text-align: center;">If you didn't create an account, you can safely ignore this email.</p>
			</div>
			<p style="color: #cbd5e1; font-size: 11px; text-align: center; margin-top: 20px;">&copy; 2026 Locolive. All rights reserved.</p>
		</div>
	`, otp)

	msg := []byte(subject + mime + body)
	auth := smtp.PlainAuth("", m.senderEmail, m.senderPassword, m.host)

	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	err := smtp.SendMail(addr, auth, m.senderEmail, []string{toEmail}, msg)
	if err != nil {
		fmt.Printf("[EMAIL ERROR] Failed to send OTP email to %s: %v\n", toEmail, err)
		return fmt.Errorf("failed to send OTP email via SMTP: %w", err)
	}

	fmt.Printf("[EMAIL SUCCESS] Sent OTP email to %s\n", toEmail)
	return nil
}

// Resend API Implementation (no SMTP needed)
type ResendMailer struct {
	apiKey      string
	fromAddress string
	frontendURL string
}

func NewResendMailer(apiKey, fromAddress, frontendURL string) Mailer {
	return &ResendMailer{
		apiKey:      apiKey,
		fromAddress: fromAddress,
		frontendURL: frontendURL,
	}
}

type resendPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

func (m *ResendMailer) send(email, subject, html string) error {
	payload := resendPayload{
		From:    m.fromAddress,
		To:      email,
		Subject: subject,
		HTML:    html,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send via Resend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, string(errBody))
	}

	return nil
}

func (m *ResendMailer) SendResetEmail(toEmail string, token string) error {
	if m.apiKey == "" {
		fmt.Printf("------------\n[DEVELOPMENT MODE - EMAIL LOG]\nTo: %s\nSubject: Password Reset\nLink: %s/reset-password?token=%s\n------------\n", toEmail, m.frontendURL, token)
		return nil
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", m.frontendURL, token)
	subject := "Reset Your Locolive Password"
	html := fmt.Sprintf(`
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

	err := m.send(toEmail, subject, html)
	if err != nil {
		fmt.Printf("[EMAIL ERROR] Failed to send reset email to %s: %v\n", toEmail, err)
		return err
	}
	fmt.Printf("[EMAIL SUCCESS] Sent reset email to %s\n", toEmail)
	return nil
}

func (m *ResendMailer) SendVerificationEmail(toEmail string, token string) error {
	if m.apiKey == "" {
		fmt.Printf("------------\n[DEVELOPMENT MODE - EMAIL LOG]\nTo: %s\nSubject: Verify Your Email\nLink: %s/verify-email?token=%s\n------------\n", toEmail, m.frontendURL, token)
		return nil
	}

	verificationLink := fmt.Sprintf("%s/verify-email?token=%s", m.frontendURL, token)
	subject := "Verify Your Locolive Email"
	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: auto; padding: 20px; border: 1px solid #eee; border-radius: 10px;">
			<div style="text-align: center; margin-bottom: 20px;">
				<h1 style="color: #FF3B8E; margin: 0; font-size: 28px;">Locolive</h1>
			</div>
			<div style="background: #f8fafc; padding: 30px; border-radius: 20px;">
				<h2 style="color: #1e293b; margin-top: 0;">Verify Your Email Address</h2>
				<p style="color: #475569; font-size: 16px; line-height: 1.6;">Welcome to Locolive! Please verify your email address to activate your account.</p>
				<p style="color: #475569; font-size: 16px; line-height: 1.6;">Click the button below to verify. This link expires in 24 hours.</p>
				<div style="text-align: center; margin: 35px 0;">
					<a href="%s" style="background: linear-gradient(to right, #FF3B8E, #A855F7); color: white; padding: 14px 32px; text-decoration: none; border-radius: 14px; font-weight: bold; font-size: 16px; display: inline-block; box-shadow: 0 10px 15px -3px rgba(255, 59, 142, 0.3);">Verify Email</a>
				</div>
				<p style="color: #94a3b8; font-size: 13px; text-align: center;">If you didn't create an account, you can safely ignore this email.</p>
			</div>
			<p style="color: #cbd5e1; font-size: 11px; text-align: center; margin-top: 20px;">&copy; 2026 Locolive. All rights reserved.</p>
		</div>
	`, verificationLink)

	err := m.send(toEmail, subject, html)
	if err != nil {
		fmt.Printf("[EMAIL ERROR] Failed to send verification email to %s: %v\n", toEmail, err)
		return err
	}
	fmt.Printf("[EMAIL SUCCESS] Sent verification email to %s\n", toEmail)
	return nil
}

func (m *ResendMailer) SendOTPEmail(toEmail string, otp string) error {
	if m.apiKey == "" {
		fmt.Printf("------------\n[DEVELOPMENT MODE - EMAIL LOG]\nTo: %s\nSubject: Your Locolive Verification Code\nOTP: %s\n------------\n", toEmail, otp)
		return nil
	}

	subject := "Your Locolive Verification Code"
	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: auto; padding: 20px; border: 1px solid #eee; border-radius: 10px;">
			<div style="text-align: center; margin-bottom: 20px;">
				<h1 style="color: #FF3B8E; margin: 0; font-size: 28px;">Locolive</h1>
			</div>
			<div style="background: #f8fafc; padding: 30px; border-radius: 20px;">
				<h2 style="color: #1e293b; margin-top: 0;">Verify Your Account</h2>
				<p style="color: #475569; font-size: 16px; line-height: 1.6;">Welcome to Locolive! Use the code below to verify your account.</p>
				<div style="text-align: center; margin: 35px 0;">
					<div style="background: #1e293b; color: white; padding: 20px 40px; border-radius: 14px; font-size: 42px; font-weight: 900; letter-spacing: 12px; display: inline-block; font-family: monospace;">%s</div>
				</div>
				<p style="color: #475569; font-size: 16px; line-height: 1.6;">Enter this code on the verification page. It expires in 24 hours.</p>
				<p style="color: #94a3b8; font-size: 13px; text-align: center;">If you didn't create an account, you can safely ignore this email.</p>
			</div>
			<p style="color: #cbd5e1; font-size: 11px; text-align: center; margin-top: 20px;">&copy; 2026 Locolive. All rights reserved.</p>
		</div>
	`, otp)

	err := m.send(toEmail, subject, html)
	if err != nil {
		fmt.Printf("[EMAIL ERROR] Failed to send OTP email to %s: %v\n", toEmail, err)
		return err
	}
	fmt.Printf("[EMAIL SUCCESS] Sent OTP email to %s\n", toEmail)
	return nil
}
