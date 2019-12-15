package hss

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

func getMessageURLs(m *discordgo.Message) []*url.URL {
	msgs := []*url.URL{}
	parts := strings.Fields(m.Content)
	for _, part := range parts {
		if strings.HasPrefix(part, "http") {
			u, err := url.Parse(part)
			if err == nil {
				msgs = append(msgs, u)
			}
		}
	}

	return msgs
}

func getMessageAttachmentURLs(m *discordgo.Message) []*url.URL {
	urls := make([]*url.URL, 0, len(m.Attachments))
	for _, att := range m.Attachments {
		u, err := url.Parse(att.ProxyURL)
		if err != nil {
			logrus.Errorf("url parse: %v", err)
			continue
		}
		urls = append(urls, u)
	}
	return urls
}

func (b *Bot) checkURL(u *url.URL) bool {
	logrus.Debugf("check url: %v", u.String())
	resp, err := b.hc.Get(u.String())
	if err != nil {
		logrus.Errorf("check url: %v", err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("check url: unexpected status: %v", resp.Status)
		return false
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("check url: download image: %v", err)
		return false
	}
	hasher := sha256.New()
	n, err := hasher.Write(body)
	if err != nil {
		logrus.Errorf("check url: compute checksum: %v", err)
		return false
	}
	if n != len(body) {
		logrus.Error("check url: compute checksum: incomplete")
		return false
	}
	hres := fmt.Sprintf("%x", hasher.Sum(nil))
	return b.checkHash(hres)
}

func (b *Bot) checkHash(hash string) bool {
	logrus.Debugf("checking hash: %v", hash)
	for _, h := range b.badHashes {
		if hash == h {
			return true
		}
	}
	return false
}

func (b *Bot) hasAdminRole(member *discordgo.Member) bool {
	for _, role := range member.Roles {
		if role == b.adminRole {
			return true
		}
	}
	return false
}

func (b *Bot) onChanMsg(m *discordgo.Message) {
	if strings.HasPrefix(m.Content, "!") {
		if b.hasAdminRole(m.Member) {
			parts := strings.Fields(m.Content)
			switch parts[0] {
			case "!hashadd":
				found := false
				for _, h := range b.badHashes {
					if h == parts[1] {
						found = true
					}
				}
				if !found {
					b.badHashes = append(b.badHashes, parts[1])
					b.saveHashes()
				}
				err := b.s.ChannelMessageDelete(m.ChannelID, m.ID)
				if err != nil {
					logrus.Errorf("discord: delete command message: %v", err)
				}
				return
			case "!hashdel":
				for i, x := range b.badHashes {
					if x == parts[1] {
						if len(b.badHashes) == 1 {
							b.badHashes = []string{}
						} else {
							b.badHashes[i] = b.badHashes[len(b.badHashes)-1]
							b.badHashes[len(b.badHashes)-1] = ""
							b.badHashes = b.badHashes[:len(b.badHashes)-1]
						}
						break
					}
				}
				b.saveHashes()
				err := b.s.ChannelMessageDelete(m.ChannelID, m.ID)
				if err != nil {
					logrus.Errorf("discord: delete command message: %v", err)
				}
				return
			default:
				return
			}
		}
	}

	urls := getMessageURLs(m)
	urls2 := getMessageAttachmentURLs(m)
	urls = append(urls, urls2...)

	for _, u := range urls {
		if b.checkURL(u) {
			b.reactBadHash(m)
		}
	}
}

func (b *Bot) reactBadHash(m *discordgo.Message) {
	err := b.s.ChannelMessageDelete(m.ChannelID, m.ID)
	if err != nil {
		logrus.Errorf("delete bad hash message: %v", err)
	}
	_, err = b.s.ChannelMessageSend("305939630220378113", fmt.Sprintf("detected bad hash from user %v", m.Author.Mention()))
	if err != nil {
		logrus.Errorf("msg send: %v", err)
	}
	/*if err := b.s.GuildBanCreateWithReason(m.GuildID, m.Author.ID, "Detected sending forbidden image", 0); err != nil {
		logrus.Errorf("error adding guild ban: %v", err)
	}*/
}

func (b *Bot) saveHashes() {
	data, err := json.MarshalIndent(b.badHashes, "", "\t")
	if err != nil {
		logrus.Errorf("save hashes: marshal: %v", err)
		return
	}
	err = ioutil.WriteFile("hashes.json", data, 0644)
	if err != nil {
		logrus.Errorf("save hashes: %v", err)
	}
}

func (b *Bot) loadHashes() {
	data, err := ioutil.ReadFile("hashes.json")
	if err != nil {
		logrus.Errorf("load hashes: %v", err)
		return
	}
	err = json.Unmarshal(data, &b.badHashes)
	if err != nil {
		logrus.Errorf("load hashes: unmarshal: %v", err)
		return
	}
}
