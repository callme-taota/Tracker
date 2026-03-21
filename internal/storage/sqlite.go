package storage

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps SQLite connection and provides access to schema.
type DB struct {
	conn *sql.DB
}

// Source row.
type Source struct {
	ID        int64     `json:"id"`
	URL       string    `json:"url"`
	Type      string    `json:"type"`
	Config    string    `json:"config"`
	CreatedAt time.Time `json:"created_at"`
}

// Item row.
type Item struct {
	ID        int64
	SourceID  sql.NullInt64
	Title     string
	URL       string
	Content   string
	Summary   string
	Timestamp time.Time
	Raw       string
	CreatedAt time.Time
}

// Open opens a SQLite database at path and runs migrations.
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec(schemaSQL); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := migrate(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	conn.SetMaxOpenConns(8)
	conn.SetMaxIdleConns(4)
	conn.SetConnMaxLifetime(0)
	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// ListSources returns all sources.
func (db *DB) ListSources() ([]Source, error) {
	rows, err := db.conn.Query("SELECT id, url, type, config, created_at FROM sources ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Source
	for rows.Next() {
		var s Source
		err := rows.Scan(&s.ID, &s.URL, &s.Type, &s.Config, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// AddSource inserts a source and returns its ID.
func (db *DB) AddSource(url, typ, config string) (int64, error) {
	res, err := db.conn.Exec("INSERT INTO sources (url, type, config) VALUES (?, ?, ?)", url, typ, config)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// DeleteSource removes a source by ID.
func (db *DB) DeleteSource(id int64) error {
	_, err := db.conn.Exec("DELETE FROM sources WHERE id = ?", id)
	return err
}

// SaveItem inserts an item (source_id optional). Returns item ID.
func (db *DB) SaveItem(sourceID *int64, title, url, content, summary string, ts time.Time, raw string) (int64, error) {
	var sid interface{}
	if sourceID != nil {
		sid = *sourceID
	}
	res, err := db.conn.Exec(
		"INSERT INTO items (source_id, title, url, content, summary, timestamp, raw) VALUES (?, ?, ?, ?, ?, ?, ?)",
		sid, title, url, content, summary, ts, raw,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// SaveSummary inserts a summary for an item. key_points as JSON array string.
func (db *DB) SaveSummary(itemID int64, summary, keyPointsJSON string) (int64, error) {
	res, err := db.conn.Exec("INSERT INTO summaries (item_id, summary, key_points) VALUES (?, ?, ?)", itemID, summary, keyPointsJSON)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListItems returns items with limit and offset. Order by created_at desc.
func (db *DB) ListItems(limit, offset int) ([]Item, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.conn.Query(
		"SELECT id, source_id, title, url, content, summary, timestamp, raw, created_at FROM items ORDER BY created_at DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Item
	for rows.Next() {
		var i Item
		err := rows.Scan(&i.ID, &i.SourceID, &i.Title, &i.URL, &i.Content, &i.Summary, &i.Timestamp, &i.Raw, &i.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, rows.Err()
}

// CountItems returns total count of items.
func (db *DB) CountItems() (int64, error) {
	var n int64
	err := db.conn.QueryRow("SELECT COUNT(*) FROM items").Scan(&n)
	return n, err
}

// SummaryRow for API (summary + item title/url).
type SummaryRow struct {
	ID        int64     `json:"id"`
	ItemID    int64     `json:"item_id"`
	ItemTitle string    `json:"item_title"`
	ItemURL   string    `json:"item_url"`
	Summary   string    `json:"summary"`
	KeyPoints string    `json:"key_points"`
	CreatedAt time.Time `json:"created_at"`
}

// ListSummaries returns summaries with limit and offset, joined with item title/url.
func (db *DB) ListSummaries(limit, offset int) ([]SummaryRow, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.conn.Query(`
		SELECT s.id, s.item_id, i.title, i.url, s.summary, s.key_points, s.created_at
		FROM summaries s LEFT JOIN items i ON s.item_id = i.id
		ORDER BY s.created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []SummaryRow
	for rows.Next() {
		var r SummaryRow
		err := rows.Scan(&r.ID, &r.ItemID, &r.ItemTitle, &r.ItemURL, &r.Summary, &r.KeyPoints, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// CountSummaries returns total count of summaries.
func (db *DB) CountSummaries() (int64, error) {
	var n int64
	err := db.conn.QueryRow("SELECT COUNT(*) FROM summaries").Scan(&n)
	return n, err
}

// ListInterests returns all interests.
func (db *DB) ListInterests() ([]Interest, error) {
	rows, err := db.conn.Query("SELECT id, name, keywords, config, created_at FROM interests ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Interest
	for rows.Next() {
		var i Interest
		err := rows.Scan(&i.ID, &i.Name, &i.Keywords, &i.Config, &i.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, rows.Err()
}

// Interest row.
type Interest struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Keywords  string    `json:"keywords"`
	Config    string    `json:"config"`
	CreatedAt time.Time `json:"created_at"`
}

// AddInterest inserts an interest. keywords can be JSON array or comma-separated.
func (db *DB) AddInterest(name, keywords, config string) (int64, error) {
	res, err := db.conn.Exec("INSERT INTO interests (name, keywords, config) VALUES (?, ?, ?)", name, keywords, config)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// DeleteInterest removes an interest by ID.
func (db *DB) DeleteInterest(id int64) error {
	_, err := db.conn.Exec("DELETE FROM interests WHERE id = ?", id)
	return err
}

// KeyPointsToJSON marshals key points for storage.
func KeyPointsToJSON(kp []string) string {
	if len(kp) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(kp)
	return string(b)
}

// KeyPointsFromJSON unmarshals key points from storage.
func KeyPointsFromJSON(s string) ([]string, error) {
	var kp []string
	err := json.Unmarshal([]byte(s), &kp)
	return kp, err
}
