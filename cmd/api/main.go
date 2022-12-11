package main

import (
	"fmt"
	"javlonrahimov/quotes-api/config"
	"os"

	"javlonrahimov/quotes-api/internal/database"
	"javlonrahimov/quotes-api/internal/leveledlog"
	"javlonrahimov/quotes-api/internal/server"
	"javlonrahimov/quotes-api/internal/smtp"
	"javlonrahimov/quotes-api/internal/version"
)

type application struct {
	config config.Config
	db     *database.DB
	logger *leveledlog.Logger
	mailer smtp.Mailer
}

func main() {
	var cfg = config.GetConfig()

	if cfg.Version {
		fmt.Printf("version: %s\n", version.Get())
		return
	}

	logger := leveledlog.NewLogger(os.Stdout, leveledlog.LevelAll, true)

	db, err := database.New(cfg.DB.DSN, cfg.DB.Automigrate)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()

	var mailer smtp.Mailer

	if cfg.UseTelegram {
		mailer = smtp.NewTelegramSender(cfg.TelegramBot.BotToken, cfg.TelegramBot.ChannelID)
	} else {
		mailer = smtp.NewEmailSender(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
	}

	app := &application{
		config: cfg,
		db:     db,
		logger: logger,
		mailer: mailer,
	}

	logger.Info("starting server on %s (version %s)", cfg.Addr, version.Get())

	err = server.Run(cfg.Addr, app.routes(), cfg.TLS.CertFile, cfg.TLS.KeyFile)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info("server stopped")
}
