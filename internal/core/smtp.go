package core

import (
	"net/smtp"
)

func smtpPlainAuth(user, pass, host string) smtp.Auth {
	return smtp.PlainAuth("", user, pass, host)
}

func smtpSendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	return smtp.SendMail(addr, a, from, to, msg)
}
