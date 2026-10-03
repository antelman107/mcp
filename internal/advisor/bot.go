package advisor

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	startReply = "Я советник AntalyaKart. Напишите, откуда и куда поехать, или спросите про остановку и маршрут."
	failReply  = "Не получилось ответить. Попробуйте ещё раз."
)

type chatter interface {
	Reply(ctx context.Context, chatID string, prior []Message, userText string) (string, error)
}

type outbound interface {
	SendText(ctx context.Context, chatID int64, replyTo int, text string) error
	NotifyTyping(ctx context.Context, chatID int64) error
	AnswerCallback(ctx context.Context, callbackID string) error
}

// Bot is the Telegram webhook front for the advisor agent.
// It acknowledges Telegram immediately, then runs the agent.
type Bot struct {
	store  *Store
	agent  chatter
	out    outbound
	secret string
	redact []string
	mu     sync.Mutex
	seen   map[int64]struct{}
	seenQ  []int64
	path   string
}

func NewBot(store *Store, agent chatter, out outbound, webhookPath, webhookSecret string, redact []string) *Bot {
	if !strings.HasPrefix(webhookPath, "/") {
		webhookPath = "/" + webhookPath
	}
	return &Bot{
		store:  store,
		agent:  agent,
		out:    out,
		secret: webhookSecret,
		redact: redact,
		seen:   map[int64]struct{}{},
		path:   strings.TrimRight(webhookPath, "/"),
	}
}

func (b *Bot) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte("ok\n"))
		}
	})
	mux.HandleFunc(b.path+"/webhook", b.webhook)
	return mux
}

func (b *Bot) webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !secretOK(b.secret, r.Header.Get("X-Telegram-Bot-Api-Secret-Token")) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var update telegramUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if b.alreadySeen(update.UpdateID) {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusOK)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	go b.process(update)
}

func (b *Bot) process(update telegramUpdate) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	incoming, ok := update.incoming()
	if !ok {
		return
	}
	if incoming.callbackID != "" {
		if err := b.out.AnswerCallback(ctx, incoming.callbackID); err != nil {
			b.logErr("answer callback", err)
		}
	}
	if strings.TrimSpace(incoming.text) == "" {
		_ = b.out.SendText(ctx, incoming.chatID, incoming.messageID, "Напишите вопрос текстом.")
		return
	}

	chatKey := strconv.FormatInt(incoming.chatID, 10)
	prior, err := b.store.Append(chatKey, "user", incoming.text)
	if err != nil {
		b.logErr("save user message", err)
		_ = b.out.SendText(ctx, incoming.chatID, incoming.messageID, failReply)
		return
	}

	reply := startReply
	if !isStart(incoming.text) {
		_ = b.out.NotifyTyping(ctx, incoming.chatID)
		reply, err = b.agent.Reply(ctx, chatKey, prior, incoming.text)
		if err != nil {
			b.logErr("agent", err)
			reply = failReply
		}
	}
	if _, err := b.store.Append(chatKey, "assistant", reply); err != nil {
		b.logErr("save assistant message", err)
	}
	if err := b.out.SendText(ctx, incoming.chatID, incoming.messageID, reply); err != nil {
		b.logErr("send message", err)
	}
}

func (b *Bot) alreadySeen(updateID int64) bool {
	if updateID == 0 {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.seen[updateID]; ok {
		return true
	}
	b.seen[updateID] = struct{}{}
	b.seenQ = append(b.seenQ, updateID)
	if len(b.seenQ) > 1024 {
		drop := b.seenQ[0]
		b.seenQ = b.seenQ[1:]
		delete(b.seen, drop)
	}
	return false
}

func (b *Bot) logErr(prefix string, err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	for _, secret := range b.redact {
		if secret != "" {
			msg = strings.ReplaceAll(msg, secret, "[REDACTED]")
		}
	}
	log.Printf("%s: %s", prefix, msg)
}

func secretOK(want, got string) bool {
	if want == "" {
		return true
	}
	if len(want) != len(got) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}

func isStart(text string) bool {
	text = strings.TrimSpace(text)
	command, _, _ := strings.Cut(text, " ")
	command, _, _ = strings.Cut(command, "@")
	switch command {
	case "/start", "/help":
		return true
	default:
		return false
	}
}

type telegramUpdate struct {
	UpdateID      int64            `json:"update_id"`
	Message       *telegramMessage `json:"message"`
	CallbackQuery *struct {
		ID      string           `json:"id"`
		Data    string           `json:"data"`
		Message *telegramMessage `json:"message"`
	} `json:"callback_query"`
}

type telegramMessage struct {
	MessageID int    `json:"message_id"`
	Text      string `json:"text"`
	Caption   string `json:"caption"`
	Chat      struct {
		ID int64 `json:"id"`
	} `json:"chat"`
}

type incomingMessage struct {
	chatID     int64
	messageID  int
	text       string
	callbackID string
}

func (u telegramUpdate) incoming() (incomingMessage, bool) {
	if u.CallbackQuery != nil {
		msg := incomingMessage{
			callbackID: u.CallbackQuery.ID,
			text:       strings.TrimSpace(u.CallbackQuery.Data),
		}
		if u.CallbackQuery.Message != nil {
			msg.chatID = u.CallbackQuery.Message.Chat.ID
			msg.messageID = u.CallbackQuery.Message.MessageID
		}
		if msg.chatID == 0 {
			return incomingMessage{}, false
		}
		return msg, true
	}
	if u.Message == nil || u.Message.Chat.ID == 0 {
		return incomingMessage{}, false
	}
	text := strings.TrimSpace(u.Message.Text)
	if text == "" {
		text = strings.TrimSpace(u.Message.Caption)
	}
	return incomingMessage{
		chatID:    u.Message.Chat.ID,
		messageID: u.Message.MessageID,
		text:      text,
	}, true
}
