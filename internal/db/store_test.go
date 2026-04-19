package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuryAndDig(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	burial, err := store.Bury(BuryInput{
		ErrorText: "TypeError: Cannot read properties of undefined (reading 'map')",
		FixText:   "added optional chaining: data?.map(...)",
		Context:   "components/UserList.tsx",
		Tags:      "react,async",
	})
	if err != nil {
		t.Fatal(err)
	}
	if burial.ID == 0 {
		t.Error("expected non-zero ID")
	}

	// dedup: same error should fail
	_, err = store.Bury(BuryInput{
		ErrorText: "TypeError: Cannot read properties of undefined (reading 'map')",
		FixText:   "different fix",
	})
	if err == nil {
		t.Error("expected dedup error, got nil")
	}

	// get all
	burials, err := store.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(burials) != 1 {
		t.Errorf("expected 1 burial, got %d", len(burials))
	}

	// update dig count
	if err := store.UpdateDigCount(burial.ID); err != nil {
		t.Fatal(err)
	}
	b, err := store.GetByID(burial.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b.TimesDug != 1 {
		t.Errorf("expected times_dug=1, got %d", b.TimesDug)
	}

	_ = os.RemoveAll(dir)
}

func TestFTSSearch(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "fts.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, _ = store.Bury(BuryInput{ErrorText: "nil pointer dereference in main.go", FixText: "add nil check", Tags: "go"})
	_, _ = store.Bury(BuryInput{ErrorText: "index out of range slice", FixText: "bounds check", Tags: "go"})

	results, err := store.FTSSearch(`"nil pointer"`, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("expected FTS result, got none")
	}
}
