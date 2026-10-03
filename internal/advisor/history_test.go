package advisor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBesideExecutableIgnoresWorkingDirectory(t *testing.T) {
	exeDir := t.TempDir()
	exe := filepath.Join(exeDir, "antalyakart-advisor")
	other := t.TempDir()
	t.Chdir(other)

	got := ResolveBesideExecutable(exe, "")
	want := filepath.Join(exeDir, "chat-history.json")
	if got != want {
		t.Fatalf("default path = %s, want %s", got, want)
	}

	rel := ResolveBesideExecutable(exe, "data/chats.json")
	if rel != filepath.Join(exeDir, "data", "chats.json") {
		t.Fatalf("relative path = %s", rel)
	}

	abs := filepath.Join(other, "absolute.json")
	if ResolveBesideExecutable(exe, abs) != abs {
		t.Fatal("absolute path was rewritten")
	}
}

func TestStoreSeparatesChatsAndReloads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chat-history.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Append("10", "user", "otogar"); err != nil {
		t.Fatal(err)
	}
	prior, err := store.Append("20", "user", "markantalya")
	if err != nil {
		t.Fatal(err)
	}
	if len(prior) != 0 {
		t.Fatalf("chat 20 prior = %+v", prior)
	}
	if _, err := store.Append("10", "assistant", "route 106"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temp history file left behind: %v", err)
	}

	reloaded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	chat10 := reloaded.Messages("10")
	chat20 := reloaded.Messages("20")
	if len(chat10) != 2 || chat10[0].Text != "otogar" || chat10[1].Role != "assistant" {
		t.Fatalf("chat 10 = %+v", chat10)
	}
	if len(chat20) != 1 || chat20[0].Text != "markantalya" {
		t.Fatalf("chat 20 = %+v", chat20)
	}
	if len(reloaded.Messages("missing")) != 0 {
		t.Fatal("missing chat was not empty")
	}
}

func TestOpenRejectsCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chat-history.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err == nil {
		t.Fatal("corrupt history was accepted")
	}
}
