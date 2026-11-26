package main

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

type smtpMailer struct {
	addr     string
	from     string
	username string
	password string
	host     string
}

func newSMTPMailer(host, port, from, username, password string) *smtpMailer {
	return &smtpMailer{
		addr:     host + ":" + port,
		host:     host,
		from:     from,
		username: username,
		password: password,
	}
}

func (m *smtpMailer) send(to, subject, body string) error {
	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	msg := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		"Subject: " + subject,
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	return smtp.SendMail(m.addr, auth, m.from, []string{to}, []byte(msg))
}

func (m *smtpMailer) SendConfirmation(to, firstName, link string, ttl time.Duration) error {
	body := fmt.Sprintf(
		"Hi %s,\r\n\r\n"+
			"Thanks for registering. Confirm your account by opening this link:\r\n\r\n"+
			"%s\r\n\r\n"+
			"The link is valid for %s. Your account stays inactive until it is confirmed.\r\n\r\n"+
			"If you did not create this account, you can ignore this email.\r\n",
		firstName, link, ttl,
	)
	return m.send(to, "Confirm your account", body)
}

func (m *smtpMailer) SendLoginCode(to, firstName, code string, ttl time.Duration) error {
	body := fmt.Sprintf(
		"Hi %s,\r\n\r\n"+
			"Your one-time login code is:\r\n\r\n"+
			"    %s\r\n\r\n"+
			"It expires in %s and can be used only once.\r\n\r\n"+
			"If you did not try to sign in, someone may know your password. Consider changing it.\r\n",
		firstName, code, ttl,
	)
	return m.send(to, "Your login code: "+code, body)
}

func (m *smtpMailer) SendPasswordReset(to, firstName, link string, ttl time.Duration) error {
	body := fmt.Sprintf(
		"Hi %s,\r\n\r\n"+
			"We received a request to reset the password of your account. Choose a new password by opening this link:\r\n\r\n"+
			"%s\r\n\r\n"+
			"The link is valid for %s and can be used only once.\r\n\r\n"+
			"If you did not request this, you can ignore this email. Your password will not change.\r\n",
		firstName, link, ttl,
	)
	return m.send(to, "Reset your password", body)
}