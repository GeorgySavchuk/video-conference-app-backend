package controllers

import (
	"fmt"
	"html"
	"log"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	apimail "github.com/GeorgySavchuk/video-conference-app-backend/mail"
	"github.com/GeorgySavchuk/video-conference-app-backend/meetingtext"
	"github.com/GeorgySavchuk/video-conference-app-backend/models"
)

var ruMonthsGenitive = []string{
	"", "января", "февраля", "марта", "апреля", "мая", "июня",
	"июля", "августа", "сентября", "октября", "ноября", "декабря",
}

var ruWeekdayLong = []string{
	"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота",
}

func capitalizeFirstRU(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func formatGMTOffset(offSec int) string {
	if offSec == 0 {
		return "GMT+00:00"
	}
	sign := "+"
	if offSec < 0 {
		sign = "-"
		offSec = -offSec
	}
	h := offSec / 3600
	m := (offSec % 3600) / 60
	return fmt.Sprintf("GMT%s%02d:%02d", sign, h, m)
}

func meetingStartLocal(m models.Meeting) (time.Time, error) {
	d, err := time.ParseInLocation("02.01.2006", strings.TrimSpace(m.Date), time.Local)
	if err != nil {
		return time.Time{}, err
	}
	tm, err := time.Parse("15:04", strings.TrimSpace(m.StartTime))
	if err != nil {
		return time.Time{}, err
	}
	h, mi, se := tm.Clock()
	return time.Date(d.Year(), d.Month(), d.Day(), h, mi, se, 0, time.Local), nil
}

// Строка вида: «Среда, 29 апреля 2026, с 12:30 до 13:30 (GMT+03:00)» — как в календарных письмах.
func formatWhenRangeRu(m models.Meeting) string {
	start, err := meetingStartLocal(m)
	if err != nil {
		return strings.TrimSpace(m.Date) + ", " + strings.TrimSpace(m.StartTime)
	}
	_, offSec := start.Zone()
	zone := formatGMTOffset(offSec)
	wd := capitalizeFirstRU(ruWeekdayLong[int(start.Weekday())])
	month := ruMonthsGenitive[int(start.Month())]
	day := start.Day()
	year := start.Year()

	if m.Duration <= 0 {
		return fmt.Sprintf("%s, %d %s %d, в %s (%s)", wd, day, month, year, start.Format("15:04"), zone)
	}
	end := start.Add(time.Duration(m.Duration) * time.Minute)
	return fmt.Sprintf("%s, %d %s %d, с %s до %s (%s)",
		wd, day, month, year, start.Format("15:04"), end.Format("15:04"), zone)
}

func buildMeetingCancelledSubjectBody(m models.Meeting) (subject string, bodyPlain string, bodyHTML string) {
	subject = "HellConf — встреча отменена"
	whenLine := formatWhenRangeRu(m)
	whenShort := strings.TrimSpace(m.Date) + " в " + strings.TrimSpace(m.StartTime)
	heading, extra := meetingtext.SplitHeading(m.Description)

	writeQuotedTitle := heading != "" && (extra != "" || utf8.RuneCountInString(heading) <= 160)

	// ----- plain -----
	var p strings.Builder
	p.WriteString("Здравствуйте!\n\n")

	if writeQuotedTitle {
		p.WriteString("Организатор отменил встречу «")
		p.WriteString(heading)
		p.WriteString("», запланированную на ")
		p.WriteString(whenShort)
		p.WriteString(".\n\n")
	} else if heading != "" {
		p.WriteString("Организатор отменил видеовстречу, запланированную на ")
		p.WriteString(whenShort)
		p.WriteString(".\n\n")
		p.WriteString("Комментарий организатора:\n")
		p.WriteString(heading)
		p.WriteString("\n\n")
	} else {
		p.WriteString("Организатор отменил видеовстречу, запланированную на ")
		p.WriteString(whenShort)
		p.WriteString(".\n\n")
	}

	p.WriteString("Когда (было запланировано): ")
	p.WriteString(whenLine)
	p.WriteString("\n\n")

	p.WriteString("Напоминания по ней отключены — писем о начале больше не будет.")

	if extra != "" {
		p.WriteString("\n\n")
		p.WriteString(extra)
	}
	p.WriteString("\n\n— HellConf\n")
	bodyPlain = p.String()

	// ----- HTML -----
	org := apimail.ReplyFromAddress()
	bodyHTML = buildCancelledEmailHTML(writeQuotedTitle, heading, extra, whenLine, org, m.Link)
	return subject, bodyPlain, bodyHTML
}

func buildCancelledEmailHTML(writeQuotedTitle bool, heading, extra, whenLine, organizerEmail, link string) string {
	hBrand := "#2563EB"
	hMuted := "#6B7280"
	hText := "#111827"
	boxBg := "#F3F4F6"

	var titleHTML strings.Builder
	if writeQuotedTitle && heading != "" {
		titleHTML.WriteString("Организатор отменил встречу «")
		titleHTML.WriteString(html.EscapeString(heading))
		titleHTML.WriteString("»")
	} else if heading != "" {
		titleHTML.WriteString("Организатор отменил видеовстречу")
	} else {
		titleHTML.WriteString("Организатор отменил видеовстречу")
	}

	var blocks strings.Builder
	blocks.WriteString(fmt.Sprintf(
		`<p style="margin:0 0 6px;font-weight:700;color:%s;font-size:15px;">Когда</p>`+
			`<p style="margin:0 0 22px;color:#374151;font-size:15px;line-height:1.55;">%s</p>`,
		hText, html.EscapeString(whenLine)))

	blocks.WriteString(fmt.Sprintf(
		`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:%s;border-radius:8px;margin:0 0 20px;"><tr><td style="padding:16px 18px;">`+
			`<p style="margin:0 0 6px;font-weight:700;color:%s;font-size:15px;">Напоминания</p>`+
			`<p style="margin:0;color:#374151;font-size:14px;line-height:1.5;">Напоминания по этой встрече отключены — писем о начале больше не будет.</p>`+
			`</td></tr></table>`,
		boxBg, hText))

	if strings.TrimSpace(link) != "" {
		escLink := html.EscapeString(strings.TrimSpace(link))
		blocks.WriteString(fmt.Sprintf(
			`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:%s;border-radius:8px;margin:0 0 20px;"><tr><td style="padding:16px 18px;">`+
				`<p style="margin:0 0 8px;font-weight:700;color:%s;font-size:15px;">Ссылка на комнату</p>`+
				`<p style="margin:0;font-size:14px;"><a href="%s" style="color:%s;text-decoration:underline;word-break:break-all;">%s</a></p>`+
				`<p style="margin:8px 0 0;color:#6B7280;font-size:13px;">Встреча отменена; ссылка приведена для справки.</p>`+
				`</td></tr></table>`,
			boxBg, hText, escLink, hBrand, escLink))
	}

	if !writeQuotedTitle && heading != "" {
		blocks.WriteString(fmt.Sprintf(
			`<p style="margin:0 0 8px;font-weight:700;color:%s;font-size:15px;">Комментарий организатора</p>`+
				`<p style="margin:0 0 20px;color:#374151;font-size:15px;line-height:1.55;white-space:pre-wrap;">%s</p>`,
			hText, html.EscapeString(heading)))
	}

	if extra != "" {
		blocks.WriteString(fmt.Sprintf(
			`<p style="margin:0 0 8px;font-weight:700;color:%s;font-size:15px;">Дополнительно</p>`+
				`<p style="margin:0 0 20px;color:#374151;font-size:15px;line-height:1.55;white-space:pre-wrap;">%s</p>`,
			hText, html.EscapeString(extra)))
	}

	orgBlock := ""
	if organizerEmail != "" {
		e := html.EscapeString(organizerEmail)
		orgBlock = fmt.Sprintf(
			`<p style="margin:0 0 6px;font-weight:700;color:%s;font-size:15px;">Организатор</p>`+
				`<p style="margin:0;font-size:15px;"><a href="mailto:%s" style="color:%s;text-decoration:underline;">%s</a></p>`,
			hText, e, hBrand, e)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="ru">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#F3F4F6;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#F3F4F6;padding:28px 16px;">
<tr><td align="center">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%%;background:#FFFFFF;border-radius:10px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,.06);">
<tr><td style="padding:26px 28px 8px;text-align:center;font-size:22px;font-weight:700;color:%s;letter-spacing:-0.02em;">HellConf</td></tr>
<tr><td style="padding:8px 28px 8px;">
<h1 style="margin:0 0 22px;font-size:22px;line-height:1.35;color:%s;font-weight:700;">%s</h1>
%s
%s
</td></tr>
<tr><td style="padding:18px 28px 22px;border-top:1px solid #E5E7EB;background:#FAFAFA;">
<p style="margin:0 0 10px;font-size:12px;line-height:1.6;color:%s;">Отправлено через <a href="#" style="color:%s;text-decoration:none;font-weight:600;">HellConf</a></p>
<p style="margin:0;font-size:12px;line-height:1.6;color:#9CA3AF;">Вы получили это письмо, так как были подписаны на напоминание об этой встрече.</p>
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>`,
		hBrand, hText, titleHTML.String(), blocks.String(), orgBlock, hMuted, hBrand)
}

// NotifySubscribersMeetingCancelled рассылает письмо об отмене по всем строкам meeting_reminders для встречи.
func NotifySubscribersMeetingCancelled(m models.Meeting) (sent int, failed []string) {
	var subs []models.MeetingReminder
	if err := initializers.DB.Where("meeting_id = ?", m.ID).Find(&subs).Error; err != nil {
		log.Printf("meeting cancel notify: list reminders: %v", err)
		return 0, nil
	}
	if len(subs) == 0 {
		return 0, nil
	}
	subj, plain, htmlBody := buildMeetingCancelledSubjectBody(m)
	for _, sub := range subs {
		email := strings.TrimSpace(sub.Email)
		if email == "" {
			continue
		}
		if err := apimail.SendMeetingCancelled(email, subj, plain, htmlBody); err != nil {
			if err == apimail.ReminderDisabled {
				continue
			}
			log.Printf("meeting cancel notify to %s: %v", email, err)
			failed = append(failed, email)
			continue
		}
		sent++
	}
	return sent, failed
}
