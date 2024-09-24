package telegram

import (
	"TelegramBot/clients/telegram"
	"TelegramBot/events"
	"TelegramBot/lib/e"
	"TelegramBot/storage"
	"errors"
	"time"
)

type Processor struct {
	tg                  *telegram.Client
	offset              int
	storage             storage.Storage
	spamProtection      map[int]time.Time
	canSendMessage      bool
	spamProtectionCount map[int]int
}

type Meta struct {
	chatID   int
	Username string
}

var (
	ErrUnknownEventType = errors.New("unknown event type")
	ErrUnknownMetaType  = errors.New("unknown meta type")
)

func New(client *telegram.Client, storage storage.Storage) *Processor {
	return &Processor{
		tg:                  client,
		storage:             storage,
		spamProtection:      make(map[int]time.Time),
		spamProtectionCount: make(map[int]int),
	}
}

func (p *Processor) Fetch(limit, offset int) ([]events.Event, error) {
	updates, err := p.tg.Updates(p.offset, limit)
	if err != nil {
		return nil, e.Wrap("cant get events", err)
	}
	if len(updates) == 0 {
		return nil, nil
	}
	res := make([]events.Event, 0, len(updates))

	for _, u := range updates {
		res = append(res, event(u))
	}

	p.offset = updates[len(updates)-1].ID + 1

	return res, nil
}

func (p *Processor) Process(event events.Event) error {
	switch event.Type {
	case events.Message:
		meta, err := meta(event)
		if err != nil {
			return e.Wrap("cant process message", err)
		}

		if err := p.doCmd(event.Text, meta.chatID, meta.Username); err != nil {
			return e.Wrap("cant process message", err)
		}
		return nil
	default:
		return e.Wrap("cant process message", ErrUnknownEventType)
	}
}

func (p *Processor) processMessage(event events.Event) error {
	meta, err := meta(event)
	if err != nil {
		return e.Wrap("cant process message", err)
	}

	if err = p.doCmd(event.Text, meta.chatID, meta.Username); err != nil {
		return e.Wrap("cant process message", err)
	}
	return nil
}

func meta(event events.Event) (Meta, error) {
	res, ok := event.Meta.(Meta)
	if !ok {
		return Meta{}, e.Wrap("cant get meta", ErrUnknownMetaType)
	}
	return res, nil
}

func event(upd telegram.Update) events.Event {
	updType := fetchType(upd)
	res := events.Event{
		Type: updType,
		Text: fetchText(upd),
	}
	if updType == events.Message {
		res.Meta = Meta{
			chatID:   upd.Message.Chat.ID,
			Username: upd.Message.From.Username,
		}
	}
	return res
}

func fetchText(upd telegram.Update) string {
	if upd.Message == nil {
		return ""
	}
	return upd.Message.Text
}

func fetchType(upd telegram.Update) events.Type {
	if upd.Message == nil {
		return events.Unknown
	}
	return events.Message
}
