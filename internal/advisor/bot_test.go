package advisor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type scriptedAgent struct {
	reply   string
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *scriptedAgent) Reply(context.Context, string, []Message, string) (string, error) {
	s.once.Do(func() { close(s.entered) })
	if s.release != nil {
		<-s.release
	}
	return s.reply, nil
}

type recordingOut struct {
	mu    sync.Mutex
	texts []string
}

func (r *recordingOut) SendText(context.Context, int64, int, string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.texts = append(r.texts, "send")
	return nil
}

func (r *recordingOut) NotifyTyping(context.Context, int64) error { return nil }
func (r *recordingOut) AnswerCallback(context.Context, string) error {
	return nil
}

func TestWebhookAcksBeforeAgentAndKeepsChatsSeparate(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir + "/chat-history.json")
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	agent := &scriptedAgent{reply: "автобус 106", entered: make(chan struct{}), release: release}
	bot := NewBot(store, agent, &recordingOut{}, "/antalyakart-advisor", "topsecret", nil)
	server := httptest.NewServer(bot.Handler())
	defer server.Close()

	body := `{"update_id":7,"message":{"message_id":3,"text":"как доехать до otogar","chat":{"id":10}}}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/antalyakart-advisor/webhook", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "topsecret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	select {
	case <-agent.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("agent was not started")
	}
	if got := store.Messages("10"); len(got) != 1 || got[0].Role != "user" {
		t.Fatalf("history before agent finished = %+v", got)
	}
	close(release)

	waitFor(t, func() bool { return len(store.Messages("10")) == 2 })
	if store.Messages("10")[1].Text != "автобус 106" {
		t.Fatalf("assistant text = %+v", store.Messages("10"))
	}

	other := `{"update_id":8,"message":{"message_id":4,"text":"/start","chat":{"id":11}}}`
	req, err = http.NewRequest(http.MethodPost, server.URL+"/antalyakart-advisor/webhook", strings.NewReader(other))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "topsecret")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	waitFor(t, func() bool { return len(store.Messages("11")) == 2 })
	if store.Messages("11")[1].Text != startReply {
		t.Fatalf("start reply = %+v", store.Messages("11"))
	}
	if len(store.Messages("10")) != 2 {
		t.Fatal("chat 10 changed while chat 11 was updated")
	}

	reloaded, err := Open(dir + "/chat-history.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Messages("10")) != 2 || len(reloaded.Messages("11")) != 2 {
		t.Fatal("reloaded history dropped a chat")
	}
}

func TestWebhookRejectsBadSecret(t *testing.T) {
	store, err := Open(t.TempDir() + "/chat-history.json")
	if err != nil {
		t.Fatal(err)
	}
	bot := NewBot(store, &scriptedAgent{reply: "x", entered: make(chan struct{})}, &recordingOut{}, "/hook", "topsecret", nil)
	req := httptest.NewRequest(http.MethodPost, "/hook/webhook", strings.NewReader(`{"update_id":1,"message":{"message_id":1,"text":"hi","chat":{"id":1}}}`))
	rec := httptest.NewRecorder()
	bot.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	if len(store.Messages("1")) != 0 {
		t.Fatal("rejected update was stored")
	}
}

func waitFor(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting")
}
