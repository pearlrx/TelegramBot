package telegram

import (
	"TelegramBot/lib/e"
	"TelegramBot/storage"
	"errors"
	"log"
	"net/url"
	"strings"
	"time"
)

const (
	RndCmd   = "/rnd"
	HelpCmd  = "/help"
	StartCmd = "/start"
)

func (p *Processor) doCmd(text string, chatID int, username string) error {
	text = strings.TrimSpace(text)

	log.Printf("got new command '%s' from '%s'", text, username)

	// Проверяем, если команда для добавления страницы
	if isAddCmd(text) {
		return p.savePage(chatID, text, username)
	}

	// Обрабатываем команду с защитой от спама
	switch text {
	case StartCmd:
		if p.isSpam(chatID) {
			return p.tg.SendMessage(chatID, "Слишком частые запросы. Попробуйте позже.")
		}
		return p.registerUser(chatID, username)
	case RndCmd:
		if p.isSpam(chatID) {
			return p.tg.SendMessage(chatID, "Слишком частые запросы. Попробуйте позже.")
		}
		return p.sendRandom(chatID, username)
	case HelpCmd:
		if p.isSpam(chatID) {
			return p.tg.SendMessage(chatID, "Слишком частые запросы. Попробуйте позже.")
		}
		return p.SendHelp(chatID)
	default:
		if p.isSpam(chatID) {
			return p.tg.SendMessage(chatID, "Слишком частые запросы. Попробуйте позже.")
		}
		return p.tg.SendMessage(chatID, msgUnknownCommand)
	}
}

func (p *Processor) registerUser(chatID int, username string) error {
	exists, err := p.storage.UserExists(username)
	if err != nil {
		return e.Wrap("can't check if user exists", err)
	}

	if exists {
		return p.SendHello(chatID)
	}

	_, err = p.storage.CreateUser(username)
	if err != nil {
		return e.Wrap("can't register user", err)
	}

	return p.SendHello(chatID)
}

func (p *Processor) savePage(chatID int, pageURL string, username string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: save message", err) }()

	page := &storage.Page{
		URL:      pageURL,
		UserName: username,
	}
	isExist, err := p.storage.IsExists(page)
	if err != nil {
		return err
	}
	if isExist {
		return p.tg.SendMessage(chatID, msgAlreadyExists)
	}

	if err := p.storage.Save(page); err != nil {
		return err
	}

	if err := p.tg.SendMessage(chatID, msgSaved); err != nil {
		return err
	}
	return nil
}

func (p *Processor) sendRandom(chatID int, username string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: can't send random", err) }()

	page, err := p.storage.PickRandom(username)
	if err != nil && !errors.Is(err, storage.ErrNoSavedPages) {
		return err
	}
	if errors.Is(err, storage.ErrNoSavedPages) {
		return p.tg.SendMessage(chatID, msgNoSavedPages)
	}

	if err := p.tg.SendMessage(chatID, page.URL); err != nil {
		return err
	}

	return p.storage.Remove(page)
}

func (p *Processor) isSpam(chatID int) bool {
	log.Printf("Checking for spam: chatID %d", chatID)
	lastMessageTime, found := p.spamProtection[chatID]

	if !found {
		p.spamProtection[chatID] = time.Now()
		log.Printf("Not found. Setting lastMessageTime to now for chatID %d", chatID)
		return false
	}

	if time.Since(lastMessageTime) < 500*time.Millisecond {
		log.Printf("Spam detected for chatID %d", chatID)
		return true
	}

	p.spamProtection[chatID] = time.Now()
	log.Printf("Spam protection passed for chatID %d", chatID)
	return false
}

func (p *Processor) SendHelp(chatID int) error {
	return p.tg.SendMessage(chatID, msgHelp)
}

func (p *Processor) SendHello(chatID int) error {
	return p.tg.SendMessage(chatID, msgHello)
}

func isAddCmd(text string) bool {
	return isURL(text)
}

func isURL(text string) bool {
	u, err := url.Parse(text)

	return err == nil && u.Host != ""
}
