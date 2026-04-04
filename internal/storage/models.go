package storage

import (
	"database/sql"
	"time"
)

type ormSource struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	URL       string    `gorm:"column:url;not null"`
	Type      string    `gorm:"column:type;not null"`
	Config    string    `gorm:"column:config"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (ormSource) TableName() string { return "sources" }

type ormItem struct {
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

func (ormItem) TableName() string { return "items" }

type ormSummary struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	ItemID    int64     `gorm:"column:item_id;not null;index:idx_summaries_item_id"`
	Summary   string    `gorm:"column:summary"`
	KeyPoints string    `gorm:"column:key_points"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (ormSummary) TableName() string { return "summaries" }

type ormInterest struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string    `gorm:"column:name;not null"`
	Keywords  string    `gorm:"column:keywords"`
	Config    string    `gorm:"column:config"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (ormInterest) TableName() string { return "interests" }

type ormPipelineDefinition struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string    `gorm:"column:name;not null"`
	GraphJSON string    `gorm:"column:graph_json;not null"`
	IsDefault bool      `gorm:"column:is_default;not null;default:false;index:idx_pipeline_definitions_default"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoCreateTime;autoUpdateTime"`
}

func (ormPipelineDefinition) TableName() string { return "pipeline_definitions" }

type ormJob struct {
	ID          int64      `gorm:"column:id;primaryKey;autoIncrement"`
	PipelineID  int64      `gorm:"column:pipeline_id;not null;index:idx_jobs_pipeline"`
	Kind        string     `gorm:"column:kind;not null;default:run"`
	Status      string     `gorm:"column:status;not null;default:pending;index:idx_jobs_status"`
	Payload     string     `gorm:"column:payload"`
	Attempt     int        `gorm:"column:attempt;not null;default:0"`
	MaxAttempt  int        `gorm:"column:max_attempt;not null;default:3"`
	Error       string     `gorm:"column:error"`
	ParentJobID *int64     `gorm:"column:parent_job_id"`
	ClaimedBy   string     `gorm:"column:claimed_by"`
	ClaimedAt   *time.Time `gorm:"column:claimed_at"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoCreateTime;autoUpdateTime"`
}

func (ormJob) TableName() string { return "jobs" }

func toPublicJob(row ormJob) Job {
	out := Job{
		ID:          row.ID,
		PipelineID:  row.PipelineID,
		Kind:        row.Kind,
		Status:      row.Status,
		Payload:     row.Payload,
		Attempt:     row.Attempt,
		MaxAttempt:  row.MaxAttempt,
		Error:       row.Error,
		ParentJobID: row.ParentJobID,
		ClaimedBy:   row.ClaimedBy,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if row.ClaimedAt != nil {
		out.ClaimedAt = *row.ClaimedAt
	}
	return out
}
