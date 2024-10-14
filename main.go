package main

import (
	tgClient "TelegramBot/clients/telegram"
	event_consumer "TelegramBot/consumer/event-consumer"
	"TelegramBot/events/telegram"
	"TelegramBot/storage/postgres"
	"context"
	"flag"
	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
	"log"
	"os"
)

const (
	tgBotHost = "api.telegram.org"
	batchSize = 100
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}
	defer conn.Close(context.Background())

	storage := postgres.New(conn)

	eventsProcessor := telegram.New(
		tgClient.New(tgBotHost, mustToken()),
		storage,
	)

	log.Print("service started")

	consumer := event_consumer.New(eventsProcessor, eventsProcessor, batchSize)

	if err = consumer.Start(); err != nil {
		log.Fatal("service is stopped", err)
	}
}

func mustToken() string {
	token := flag.String(
		"tg-bot-token",
		"",
		"token for access to telegram bot",
	)
	flag.Parse()
	if *token == "" {
		log.Fatal("token is not specified")
	}
	return *token
}
