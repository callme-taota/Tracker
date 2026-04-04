package storage

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"time"

	gsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// DB wraps GORM and provides access to schema.
type DB struct {
	orm *gorm.DB
}

// Source row.
type Source struct {
	ID        int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	URL       string    `json:"url" gorm:"column:url;not null"`
	Type      string    `json:"type" gorm:"column:type;not null"`
	Config    string    `json:"config" gorm:"column:config"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (Source) TableName() string { return "sources" }

// Item row.
type Item struct {
	ID        int64         `gorm:"column:id;primaryKey;autoIncrement"`
	SourceID  sql.NullInt64 `gorm:"column:source_id"`
	Title     string        `gorm:"column:title"`
	URL       string        `gorm:"column:url"`
	Content   string        `gorm:"column:content"`
	Summary   string        `gorm:"column:summary"`
	Timestamp time.Time     `gorm:"column:timestamp"`
	Raw       string        `gorm:"column:raw"`
	CreatedAt time.Time     `gorm:"column:created_at;autoCreateTime"`
}

func (Item) TableName() string { return "items" }

// Open opens a SQLite database at path and runs migrations.
func Open(path string) (*DB, error) {
	orm, err := gorm.Open(gsqlite.Open(path), &gorm.Config{
		Logger: glogger.New(
			log.New(os.Stdout, "", log.LstdFlags),
			glogger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  glogger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := orm.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(8)
	sqlDB.SetMaxIdleConns(4)
	sqlDB.SetConnMaxLifetime(0)
	if err := migrate(orm); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return &DB{orm: orm}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	sqlDB, err := db.orm.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ListSources returns all sources.
func (db *DB) ListSources() ([]Source, error) {
	var list []Source
	err := db.orm.Order("id").Find(&list).Error
	return list, err
}

// AddSource inserts a source and returns its ID.
func (db *DB) AddSource(url, typ, config string) (int64, error) {
	row := Source{URL: url, Type: typ, Config: config}
	if err := db.orm.Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// DeleteSource removes a source by ID.
func (db *DB) DeleteSource(id int64) error {
	return db.orm.Delete(&Source{}, id).Error
}

// SaveItem inserts an item (source_id optional). Returns item ID.
func (db *DB) SaveItem(sourceID *int64, title, url, content, summary string, ts time.Time, raw string) (int64, error) {
	row := Item{
		Title:     title,
		URL:       url,
		Content:   content,
		Summary:   summary,
		Timestamp: ts,
		Raw:       raw,
	}
	if sourceID != nil {
		row.SourceID = sql.NullInt64{Int64: *sourceID, Valid: true}
	}
	if err := db.orm.Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// SaveSummary inserts a summary for an item. key_points as JSON array string.
func (db *DB) SaveSummary(itemID int64, summary, keyPointsJSON string) (int64, error) {
	row := ormSummary{ItemID: itemID, Summary: summary, KeyPoints: keyPointsJSON}
	if err := db.orm.Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// ListItems returns items with limit and offset. Order by created_at desc.
func (db *DB) ListItems(limit, offset int) ([]Item, error) {
	var list []Item
	err := applyPagination(db.orm.Order("created_at DESC"), limit, offset).Find(&list).Error
	return list, err
}

// CountItems returns total count of items.
func (db *DB) CountItems() (int64, error) {
	var n int64
	err := db.orm.Model(&Item{}).Count(&n).Error
	return n, err
}

// SummaryRow for API (summary + item title/url).
type SummaryRow struct {
	ID        int64     `json:"id" gorm:"column:id"`
	ItemID    int64     `json:"item_id" gorm:"column:item_id"`
	ItemTitle string    `json:"item_title" gorm:"column:item_title"`
	ItemURL   string    `json:"item_url" gorm:"column:item_url"`
	Summary   string    `json:"summary" gorm:"column:summary"`
	KeyPoints string    `json:"key_points" gorm:"column:key_points"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
}

// ListSummaries returns summaries with limit and offset, joined with item title/url.
func (db *DB) ListSummaries(limit, offset int) ([]SummaryRow, error) {
	var list []SummaryRow
	query := summaryRowsQuery(db.orm).Order("s.created_at DESC")
	err := applyPagination(query, limit, offset).Scan(&list).Error
	return list, err
}

// CountSummaries returns total count of summaries.
func (db *DB) CountSummaries() (int64, error) {
	var n int64
	err := db.orm.Model(&ormSummary{}).Count(&n).Error
	return n, err
}

// Interest row.
type Interest struct {
	ID        int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"column:name;not null"`
	Keywords  string    `json:"keywords" gorm:"column:keywords"`
	Config    string    `json:"config" gorm:"column:config"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (Interest) TableName() string { return "interests" }

// ListInterests returns all interests.
func (db *DB) ListInterests() ([]Interest, error) {
	var list []Interest
	err := db.orm.Order("id").Find(&list).Error
	return list, err
}

// AddInterest inserts an interest. keywords can be JSON array or comma-separated.
func (db *DB) AddInterest(name, keywords, config string) (int64, error) {
	row := Interest{Name: name, Keywords: keywords, Config: config}
	if err := db.orm.Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// DeleteInterest removes an interest by ID.
func (db *DB) DeleteInterest(id int64) error {
	return db.orm.Delete(&Interest{}, id).Error
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
