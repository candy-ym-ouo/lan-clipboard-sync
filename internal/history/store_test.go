package history

import (
	"lan-clipboard-sync/internal/core"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "history.json")
	in := []core.Message{{ID: "1", SenderID: "x", Sender: "X", Text: "text", Created: time.Now()}}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || len(got) != 1 || got[0].Text != "text" {
		t.Fatalf("%v %#v", err, got)
	}
}
