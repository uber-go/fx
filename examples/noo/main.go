package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/fx/config"
	"go.uber.org/fx/service2"
	"go.uber.org/fx/ulog"
	"go.uber.org/zap"
)

func main() {
	svc := service2.New(
		FxZapNew,
		Ticker,
	)
	svc.Start()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	log.Println(<-c)
}

// FxZapNew is a component constructor thing for zap
func FxZapNew(cfg config.Provider) (*zap.Logger, error) {
	logConfig := ulog.Configuration{}
	logConfig.Configure(cfg.Get("logging"))
	l, err := logConfig.Build()
	return l, err
}

// Ticker off
func Ticker(l *zap.Logger) *time.Ticker {
	ticker := time.NewTicker(time.Second * 1)
	go func() {
		for range ticker.C {
			l.Info("I'm alive")
		}
	}()
	return ticker
}
