package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenAndSources(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("db file not created: %v", err)
	}
	id, err := db.AddSource("https://example.com/feed", "rss", "{}")
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Errorf("AddSource want positive id got %d", id)
	}
	list, err := db.ListSources()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].URL != "https://example.com/feed" {
		t.Errorf("ListSources want 1 source got %d %v", len(list), list)
	}
	err = db.DeleteSource(id)
	if err != nil {
		t.Fatal(err)
	}
	list, _ = db.ListSources()
	if len(list) != 0 {
		t.Errorf("after Delete want 0 sources got %d", len(list))
	}
}

func TestJobsTableHasClaimColumnsAfterMigrate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "migrate.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var names []string
	cols, err := db.orm.Migrator().ColumnTypes(&ormJob{})
	if err != nil {
		t.Fatal(err)
	}
	for _, col := range cols {
		names = append(names, col.Name())
	}
	has := func(want string) bool {
		for _, n := range names {
			if n == want {
				return true
			}
		}
		return false
	}
	if !has("claimed_by") || !has("claimed_at") {
		t.Fatalf("jobs columns after migrate: want claimed_by and claimed_at, got %v", names)
	}
}

func TestMigrateIdempotentOnFreshDB(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "idempotent.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrate(db.orm); err != nil {
		t.Fatal(err)
	}
	if err := migrate(db.orm); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestItemsAndSummaries(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	itemID, err := db.SaveItem(nil, "title", "https://u", "content", "summary", time.Now(), "")
	if err != nil {
		t.Fatal(err)
	}
	if itemID <= 0 {
		t.Fatal("SaveItem want positive id")
	}
	_, err = db.SaveSummary(itemID, "sum", "[]")
	if err != nil {
		t.Fatal(err)
	}
	items, err := db.ListItems(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "title" {
		t.Errorf("ListItems want 1 got %d %v", len(items), items)
	}
	summaries, err := db.ListSummaries(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].Summary != "sum" {
		t.Errorf("ListSummaries want 1 got %d %v", len(summaries), summaries)
	}
}
