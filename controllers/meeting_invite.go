package controllers

import (
	"fmt"
	"log"
	netmail "net/mail"
	"strings"

	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	apimail "github.com/GeorgySavchuk/video-conference-app-backend/mail"
	"github.com/GeorgySavchuk/video-conference-app-backend/meetingtext"
	"github.com/GeorgySavchuk/video-conference-app-backend/models"
)

const maxInviteEmailsPerMeeting = 50

func normalizeInviteEmails(raw []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, r := range raw {
		e := strings.TrimSpace(strings.ToLower(r))
		if e == "" {
			continue
		}
		if _, err := netmail.ParseAddress(e); err != nil {
			continue
		}
		if seen[e] {
			continue
		}
		seen[e] = true
		out = append(out, e)
		if len(out) >= maxInviteEmailsPerMeeting {
			break
		}
	}
	return out
}

func buildMeetingInviteSubjectBody(m models.Meeting) (subject string, body string) {
	subject = "HellConf — приглашение на видеовстречу"
	var b strings.Builder
	b.WriteString("Здравствуйте!\n\n")
	b.WriteString("Вы приглашены на видеовстречу в HellConf.\n\n")
	b.WriteString("Дата: " + strings.TrimSpace(m.Date) + "\n")
	b.WriteString("Время начала: " + strings.TrimSpace(m.StartTime) + "\n")
	b.WriteString(fmt.Sprintf("Планируемая длительность: %d мин.\n", m.Duration))
	desc := strings.TrimSpace(m.Description)
	if desc != "" {
		heading, extra := meetingtext.SplitHeading(desc)
		switch {
		case heading != "" && extra != "":
			b.WriteString("\nНазвание встречи:\n")
			b.WriteString(heading + "\n")
			b.WriteString("\nОписание от организатора:\n")
			b.WriteString(extra + "\n")
		case heading != "":
			// Одно поле без разделителя: только название или только текст — нейтральная подпись
			b.WriteString("\nОт организатора:\n")
			b.WriteString(heading + "\n")
		}
	}
	if strings.TrimSpace(m.Link) != "" {
		b.WriteString("\nЧтобы подключиться, откройте ссылку в браузере:\n")
		b.WriteString(strings.TrimSpace(m.Link) + "\n")
	}
	b.WriteString("\nПримерно за 15 минут до начала мы отправим напоминание на этот адрес.\n")
	b.WriteString("\n— HellConf\n")
	return subject, b.String()
}

// AfterCreateMeetingInvites регистрирует участников для напоминаний и рассылает приглашения по SMTP.
func AfterCreateMeetingInvites(m models.Meeting, inviteEmails []string) (sent int, failed []string) {
	emails := normalizeInviteEmails(inviteEmails)
	if len(emails) == 0 {
		return 0, nil
	}

	for _, email := range emails {
		var existing models.MeetingReminder
		q := initializers.DB.Where("meeting_id = ? AND email = ?", m.ID, email).Limit(1).Find(&existing)
		if q.Error != nil {
			log.Printf("meeting invite check %s: %v", email, q.Error)
			failed = append(failed, email)
			continue
		}
		if q.RowsAffected > 0 {
			continue
		}

		rec := models.MeetingReminder{MeetingID: m.ID, Email: email}
		if err := initializers.DB.Create(&rec).Error; err != nil {
			log.Printf("meeting reminder insert %s: %v", email, err)
			failed = append(failed, email)
			continue
		}

		subj, body := buildMeetingInviteSubjectBody(m)
		if err := apimail.SendMeetingInvite(email, subj, body); err != nil {
			if err == apimail.ReminderDisabled {
				// SMTP не настроен — подписка на напоминание всё равно сохранена
				continue
			}
			log.Printf("invite email %s: %v", email, err)
			failed = append(failed, email)
			continue
		}
		sent++
	}
	return sent, failed
}
