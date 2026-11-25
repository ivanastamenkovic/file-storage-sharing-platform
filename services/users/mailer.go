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
