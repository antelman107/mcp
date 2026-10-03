package advisor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const telegramAPI = "https://api.telegram.org"

// TelegramClient talks to the Bot API. The token stays in memory and is not logged.
type TelegramClient struct {
	http   *http.Client
	token  string
	secret string
}

func NewTelegramClient(token, webhookSecret string) *TelegramClient {
	return &TelegramClient{
		http:   &http.Client{Timeout: 20 * time.Second},
		token:  strings.TrimSpace(token),
		secret: webhookSecret,
	}
}

func (c *TelegramClient) SetWebhook(ctx context.Context, publicURL string) error {
	body := map[string]any{
		"url":             publicURL,
		"allowed_updates": []string{"message", "callback_query"},
	}
	if c.secret != "" {
		body["secret_token"] = c.secret
	}
	return c.call(ctx, "setWebhook", body, nil)
}

func (c *TelegramClient) SendText(ctx context.Context, chatID int64, replyTo int, text string) error {
	for _, chunk := range splitTelegramText(text) {
		payload := map[string]any{
			"chat_id": chatID,
			"text":    chunk,
		}
		if replyTo != 0 {
			payload["reply_parameters"] = map[string]any{"message_id": replyTo}
			replyTo = 0
		}
		if err := c.call(ctx, "sendMessage", payload, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *TelegramClient) NotifyTyping(ctx context.Context, chatID int64) error {
	return c.call(ctx, "sendChatAction", map[string]any{
		"chat_id": chatID,
		"action":  "typing",
	}, nil)
}

func (c *TelegramClient) AnswerCallback(ctx context.Context, callbackID string) error {
	if callbackID == "" {
		return nil
	}
	return c.call(ctx, "answerCallbackQuery", map[string]any{
		"callback_query_id": callbackID,
	}, nil)
}

func (c *TelegramClient) call(ctx context.Context, method string, payload any, out any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.methodURL(method), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram %s failed: %s", method, resp.Status)
	}
	var envelope struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("telegram %s returned invalid json", method)
	}
	if !envelope.OK {
		if envelope.Description == "" {
			envelope.Description = "unknown error"
		}
		return fmt.Errorf("telegram %s rejected the call: %s", method, envelope.Description)
	}
	if out != nil && len(envelope.Result) > 0 {
		return json.Unmarshal(envelope.Result, out)
	}
	return nil
}

func (c *TelegramClient) methodURL(method string) string {
	return telegramAPI + "/bot" + c.token + "/" + method
}

func splitTelegramText(text string) []string {
	const limit = 4000
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{"..."}
	}
	if len(text) <= limit {
		return []string{text}
	}
	var chunks []string
	for len(text) > limit {
		chunks = append(chunks, text[:limit])
		text = text[limit:]
	}
	if text != "" {
		chunks = append(chunks, text)
	}
	return chunks
}
