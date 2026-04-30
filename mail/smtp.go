package mail

import (
	"bytes"
	"fmt"
	"html"
	"mime/quotedprintable"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
)

// ReminderDisabled — SMTP не настроен (локальная разработка).
var ReminderDisabled = fmt.Errorf("smtp не настроен: задайте SMTP_HOST и SMTP_FROM")

// SendMeetingInvite — приглашение при планировании.
func SendMeetingInvite(toEmail, subject, body string) error {
	return SendMeetingReminder(toEmail, subject, body)
}

// ReplyFromAddress — адрес из SMTP_FROM (часть после имён), для строки «Организатор» в письмах.
func ReplyFromAddress() string {
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		return ""
	}
	addr, err := mail.ParseAddress(from)
	if err != nil {
		return ""
	}
	return addr.Address
}

// SendMeetingCancelled отправляет только HTML (plain игнорируется при непустом bodyHTML).
func SendMeetingCancelled(toEmail, subject, bodyPlain, bodyHTML string) error {
	if strings.TrimSpace(bodyHTML) != "" {
		return SendMeetingHTML(toEmail, subject, bodyHTML)
	}
	return plainBodyAsHTML(toEmail, subject, bodyPlain)
}

// SendMeetingReminder отправляет письмо как HTML (plain текст экранируется и оборачивается в разметку).
func SendMeetingReminder(toEmail, subject, body string) error {
	return plainBodyAsHTML(toEmail, subject, body)
}

func plainBodyAsHTML(toEmail, subject, plain string) error {
	esc := html.EscapeString(plain)
	doc := `<!DOCTYPE html><html lang="ru"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head><body style="margin:0;padding:24px;font-family:system-ui,-apple-system,sans-serif;font-size:15px;line-height:1.55;color:#111827;background:#f9fafb"><div style="max-width:560px;margin:0 auto">` +
		`<pre style="white-space:pre-wrap;font-family:inherit;margin:0">` + esc + `</pre></div></body></html>`
	return SendMeetingHTML(toEmail, subject, doc)
}

// SendMeetingHTML отправляет одну часть text/html (quoted-printable).
func SendMeetingHTML(toEmail, subject, htmlBody string) error {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	pass := strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if host == "" || from == "" {
		return ReminderDisabled
	}

	var bodyBuf bytes.Buffer
	qpw := quotedprintable.NewWriter(&bodyBuf)
	if _, err := qpw.Write([]byte(htmlBody)); err != nil {
		return err
	}
	if err := qpw.Close(); err != nil {
		return err
	}

	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n",
		from, toEmail, subject))
	msg.Write(bodyBuf.Bytes())

	addr := host + ":" + port
	auth := smtp.PlainAuth("", user, pass, host)
	if user == "" && pass == "" {
		return smtp.SendMail(addr, nil, from, []string{toEmail}, msg.Bytes())
	}
	return smtp.SendMail(addr, auth, from, []string{toEmail}, msg.Bytes())
}
