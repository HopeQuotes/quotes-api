package config

import (
	"flag"
	"strings"
)

type Config struct {
	Addr    string
	BaseURL string
	Env     string
	DB      struct {
		DSN         string
		Automigrate bool
	}
	JWT struct {
		SecretKey string
	}
	SMTP struct {
		Host     string
		Port     int
		Username string
		Password string
		From     string
	}
	TelegramBot struct {
		BotToken  string
		ChannelID string
	}
	TLS struct {
		CertFile string
		KeyFile  string
	}
	Limiter struct {
		RPS     float64
		Burst   int
		Enabled bool
	}
	Cors struct {
		TrustedOrigins []string
	}
	Sudoers     []string
	Version     bool
	UseTelegram bool
}

func GetConfig() Config {

	var cfg Config

	flag.StringVar(&cfg.Addr, "addr", "localhost:4444", "server address to listen on")
	flag.StringVar(&cfg.BaseURL, "base-url", "", "base URL for the application")
	flag.StringVar(&cfg.Env, "env", "development", "operating environment: development, testing, staging or production")

	flag.StringVar(&cfg.DB.DSN, "db-dsn", "", "postgreSQL DSN")
	flag.BoolVar(&cfg.DB.Automigrate, "db-automigrate", true, "run migrations on startup")

	flag.StringVar(&cfg.JWT.SecretKey, "jwt-secret-key", "", "secret key for JWT authentication")

	flag.StringVar(&cfg.SMTP.Host, "smtp-host", "example.smtp.host", "smtp host")
	flag.IntVar(&cfg.SMTP.Port, "smtp-port", 25, "smtp port")
	flag.StringVar(&cfg.SMTP.Username, "smtp-username", "example_username", "smtp username")
	flag.StringVar(&cfg.SMTP.Password, "smtp-password", "pa55word", "smtp password")
	flag.StringVar(&cfg.SMTP.From, "smtp-from", "Example Name <no-reply@example.org>", "smtp sender")

	flag.StringVar(&cfg.TelegramBot.BotToken, "telegram-bot-token", "", "Telegram bot token")
	flag.StringVar(&cfg.TelegramBot.ChannelID, "telegram-channel-id", "", "Telegram channel id")

	flag.StringVar(&cfg.TLS.CertFile, "tls-cert-file", "./tls/cert.pem", "tls certificate file")
	flag.StringVar(&cfg.TLS.KeyFile, "tls-key-file", "./tls/key.pem", "tls key file")

	flag.Float64Var(&cfg.Limiter.RPS, "limiter-rps", 2, "rate limiter maximum requests per second")
	flag.IntVar(&cfg.Limiter.Burst, "limiter-burst", 4, "rate limiter maximum burst")
	flag.BoolVar(&cfg.Limiter.Enabled, "limiter-enabled", true, "enable rate limiter")

	flag.BoolVar(&cfg.Version, "version", false, "display version and exit")

	flag.BoolVar(&cfg.UseTelegram, "use-telegram", false, "Send OTPs through Telegram")

	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.Cors.TrustedOrigins = strings.Fields(val)
		return nil
	})

	flag.Func("sudoers", "Super user email list", func(s string) error {
		cfg.Sudoers = strings.Fields(s)
		return nil
	})

	flag.Parse()
	return cfg
}
