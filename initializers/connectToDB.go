package initializers

import (
	"log"
	"net"
	"net/url"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func databaseDSN() string {
	if dsn := os.Getenv("DB_URL"); dsn != "" {
		return dsn
	}
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "video_conference"
	}
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + dbname,
	}
	q := url.Values{}
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	return u.String()
}

func ConnectToDB() {
	dsn := databaseDSN()
	cfg := &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	}

	// Postgres и DNS Docker иногда не готовы сразу после `docker compose up` / restart docker.
	var err error
	for attempt := 1; ; attempt++ {
		DB, err = gorm.Open(postgres.Open(dsn), cfg)
		if err == nil {
			return
		}
		if attempt >= 30 {
			break
		}
		log.Printf("database: подключение не удалось (попытка %d/30): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	panic("Не удалось подключиться к базе данных: " + err.Error())
}
