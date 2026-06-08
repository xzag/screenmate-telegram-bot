package main

import (
	"flag"
	"log"

	"screenmate-bot/internal/bot"
	"screenmate-bot/internal/config"
	"screenmate-bot/internal/service"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	svc := service.New(cfg)

	b, err := bot.New(cfg, svc)
	if err != nil {
		log.Fatal(err)
	}

	b.Run()
}
