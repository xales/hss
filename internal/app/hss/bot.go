package hss

import (
	"context"
	"net/http"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

type Bot struct {
	wg        sync.WaitGroup
	apitok    string
	s         *discordgo.Session
	hc        *http.Client
	adminRole string
	badHashes []string
}

func Run(ctx context.Context, apitok, adminrole string) (*Bot, error) {
	bot := &Bot{
		apitok:    apitok,
		adminRole: adminrole,
		hc:        &http.Client{},
	}
	sess, err := discordgo.New("Bot " + apitok)
	if err != nil {
		return nil, err
	}

	bot.s = sess
	bot.s.AddHandler(func(s *discordgo.Session, _ *discordgo.Connect) {
		logrus.Info("discord: connected")
	})
	bot.s.AddHandler(func(s *discordgo.Session, msg *discordgo.MessageCreate) {
		if msg.Author.ID == bot.s.State.User.ID {
			return
		}
		if msg.GuildID != "" {
			bot.onChanMsg(msg.Message)
		}
	})

	bot.loadHashes()

	err = bot.s.Open()
	if err != nil {
		return nil, err
	}

	bot.wg.Add(1)
	go func() {
		defer bot.wg.Done()
		<-ctx.Done()
		logrus.Info("discord: shutdown")
		err := bot.s.Close()
		if err != nil {
			logrus.Errorf("discord: shutdown: %v", err)
		} else {
			logrus.Info("discord: shutdown complete")
		}
	}()

	return bot, nil
}

func (b *Bot) Wait() {
	b.wg.Wait()
}
