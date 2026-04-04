package storage

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Job is a queued pipeline execution (embedded async model).
type Job struct {
	ID          int64     `json:"id"`
	PipelineID  int64     `json:"pipeline_id"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	Payload     string    `json:"payload,omitempty"`
	Attempt     int       `json:"attempt"`
	MaxAttempt  int       `json:"max_attempt"`
	Error       string    `json:"error,omitempty"`
	ParentJobID *int64    `json:"parent_job_id,omitempty"`
	ClaimedBy   string    `json:"claimed_by,omitempty"`
	ClaimedAt   time.Time `json:"claimed_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EnqueueJob inserts a pending job.
func (db *DB) EnqueueJob(pipelineID int64, kind string, payload map[string]interface{}, maxAttempt int) (int64, error) {
	if maxAttempt <= 0 {
		maxAttempt = 3
	}
	var blob string
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return 0, err
		}
		blob = string(b)
	}
	if kind == "" {
		kind = "run"
	}
	row := ormJob{
		PipelineID: pipelineID,
		Kind:       kind,
		Status:     "pending",
		Payload:    blob,
		MaxAttempt: maxAttempt,
	}
	if err := db.orm.Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// GetJob loads a job by id.
func (db *DB) GetJob(id int64) (*Job, error) {
	var row ormJob
	err := db.orm.First(&row, id).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	out := toPublicJob(row)
	return &out, nil
}

// ClaimNextPendingJob atomically moves the oldest pending job to running and returns it.
// claimedBy should identify this process (e.g. TRACKER_INSTANCE_ID); empty becomes "default".
func (db *DB) ClaimNextPendingJob(claimedBy string) (*Job, error) {
	if claimedBy == "" {
		claimedBy = "default"
	}
	var claimed *Job
	err := db.orm.Transaction(func(tx *gorm.DB) error {
		var row ormJob
		err := tx.Where("status = ?", "pending").Order("id ASC").First(&row).Error
		if err != nil {
			if isNotFound(err) {
				return nil
			}
			return err
		}
		now := time.Now()
		res := tx.Model(&ormJob{}).
			Where("id = ? AND status = ?", row.ID, "pending").
			Updates(map[string]interface{}{
				"status":     "running",
				"attempt":    gorm.Expr("attempt + 1"),
				"claimed_by": claimedBy,
				"claimed_at": &now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		if err := tx.First(&row, row.ID).Error; err != nil {
			return err
		}
		out := toPublicJob(row)
		claimed = &out
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

// CompleteJob marks job succeeded.
func (db *DB) CompleteJob(id int64) error {
	return db.orm.Model(&ormJob{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": "done",
		"error":  "",
	}).Error
}

// FailJob marks failed; if attempt < max_attempt, reset to pending for retry.
func (db *DB) FailJob(id int64, attempt, maxAttempt int, errMsg string) error {
	if attempt < maxAttempt {
		return db.orm.Model(&ormJob{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status": "pending",
			"error":  errMsg,
		}).Error
	}
	return db.orm.Model(&ormJob{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": "failed",
		"error":  errMsg,
	}).Error
}
