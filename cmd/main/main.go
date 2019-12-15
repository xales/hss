package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/xales/hss/internal/app/hss"
)

type Config struct {
	Token     string `json:"token"`
	AdminRole string `json:"adminRole"`
	Debug     bool   `json:"debug"`
}

func signalHandler(canc context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	logrus.Infof("shutdown: received deadly signal: %v", <-c)
	canc()
}

func main() {
	configData, err := ioutil.ReadFile("config.json")
	if err != nil {
		logrus.Fatal(err)
		return
	}
	var config Config
	err = json.Unmarshal(configData, &config)
	if err != nil {
		logrus.Fatal(err)
		return
	}

	if config.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	ctx, canc := context.WithCancel(context.Background())

	go signalHandler(canc)

	bot, err := hss.Run(ctx, config.Token, config.AdminRole)
	if err != nil {
		logrus.Fatal(err)
		return
	}

	bot.Wait()
}
