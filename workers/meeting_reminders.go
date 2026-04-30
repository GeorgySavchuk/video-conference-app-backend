package workers

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	"github.com/GeorgySavchuk/video-conference-app-backend/mail"
	"github.com/GeorgySavchuk/video-conference-app-backend/meetingtext"
	"github.com/GeorgySavchuk/video-conference-app-backend/models"
	"gorm.io/gorm"
)

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

const reminderLeadBeforeStart = 15 * time.Minute

// StartMeetingReminderWorker раз в минуту проверяет подписки и шлёт письма один раз,
// когда до начала встречи осталось не больше 15 минут (и встреча ещё не началась).
func StartMeetingReminderWorker() {
	if strings.TrimSpace(os.Getenv("SMTP_HOST")) == "" || strings.TrimSpace(os.Getenv("SMTP_FROM")) == "" {
		log.Println("meeting reminders: SMTP_HOST / SMTP_FROM не заданы — письма не отправляются (задайте переменные окружения)")
	}
	ticker := time.NewTicker(time.Minute)
	go func() {
		for range ticker.C {
			runReminderBatch(initializers.DB)
		}
	}()
}

func runReminderBatch(db *gorm.DB) {
	now := time.Now()
	var subs []models.MeetingReminder
	if err := db.Where("reminder_sent_at IS NULL").Find(&subs).Error; err != nil {
		log.Printf("meeting reminders: list: %v", err)
		return
	}
	for _, sub := range subs {
		var m models.Meeting
		if err := db.First(&m, sub.MeetingID).Error; err != nil {
			continue
		}
		startAt, err := meetingStartLocal(m)
		if err != nil {
			continue
		}
		until := startAt.Sub(now)
		// Первый тик в интервале (0, 15 мин] — отправляем; reminder_sent_at исключает повторы.
		// Раньше было узкое окно 13–15 мин — легко пропустить минутный тик.
		if until <= 0 || until > reminderLeadBeforeStart {
			continue
		}
		subject := "HellConf: скоро начало встречи"
		body := "Напоминание: встреча начнётся примерно через 15 минут.\n\n"
		desc := strings.TrimSpace(m.Description)
		if desc != "" {
			h, ex := meetingtext.SplitHeading(desc)
			switch {
			case h != "" && ex != "":
				body += "Название встречи: " + h + "\n\n"
				body += "Описание: " + ex + "\n\n"
			case h != "":
				body += "От организатора:\n" + h + "\n\n"
			}
		}
		if strings.TrimSpace(m.Link) != "" {
			body += "Ссылка: " + m.Link + "\n"
		}
		if err := mail.SendMeetingReminder(sub.Email, subject, body); err != nil {
			if err == mail.ReminderDisabled {
				continue
			}
			log.Printf("meeting reminder smtp to %s: %v", sub.Email, err)
			continue
		}
		log.Printf("meeting reminder: письмо отправлено на %s (встреча id=%d)", sub.Email, m.ID)
		t := time.Now()
		_ = db.Model(&models.MeetingReminder{}).Where("id = ?", sub.ID).Update("reminder_sent_at", t).Error
	}
}
