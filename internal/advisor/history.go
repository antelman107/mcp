package advisor

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const defaultHistoryName = "chat-history.json"

// Message is one turn in a Telegram chat.
type Message struct {
	Role string `json:"role"`
	Text string `json:"text"`
	Time string `json:"time"`
}

// Store is a durable map of chat id to message history.
// The file lives beside the executable unless an absolute path is configured,
// so the working directory at startup does not matter.
type Store struct {
	mu    sync.Mutex
	path  string
	chats map[string][]Message
}

// ResolveBesideExecutable joins a relative history path to the executable's
// directory. An absolute configured path is kept as-is.
func ResolveBesideExecutable(executablePath, configured string) string {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		configured = defaultHistoryName
	}
	if filepath.IsAbs(configured) {
		return configured
	}
	return filepath.Join(filepath.Dir(executablePath), configured)
}

// DefaultHistoryPath resolves CHAT_HISTORY_PATH against the running executable.
func DefaultHistoryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return ResolveBesideExecutable(exe, os.Getenv("CHAT_HISTORY_PATH")), nil
}

// Open loads history from path. A missing file starts empty.
func Open(path string) (*Store, error) {
	store := &Store{
		path:  path,
		chats: map[string][]Message{},
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return store, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(data, &store.chats); err != nil {
		return nil, err
	}
	if store.chats == nil {
		store.chats = map[string][]Message{}
	}
	return store, nil
}

// Messages returns a copy of one chat's history.
func (s *Store) Messages(chatID string) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneMessages(s.chats[chatID])
}

// Append adds a message, writes the whole map durably, and returns the chat
// history from before this message.
func (s *Store) Append(chatID, role, text string) ([]Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prior := cloneMessages(s.chats[chatID])
	s.chats[chatID] = append(s.chats[chatID], Message{
		Role: role,
		Text: text,
		Time: time.Now().UTC().Format(time.RFC3339),
	})
	if err := s.saveLocked(); err != nil {
		s.chats[chatID] = prior
		return nil, err
	}
	return prior, nil
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.chats, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := s.path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(tmp)
		return writeErr
	}
	if syncErr != nil {
		_ = os.Remove(tmp)
		return syncErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, s.path)
}

func cloneMessages(in []Message) []Message {
	if len(in) == 0 {
		return nil
	}
	out := make([]Message, len(in))
	copy(out, in)
	return out
}
