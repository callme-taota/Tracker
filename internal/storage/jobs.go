package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
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
	res, err := db.conn.Exec(`
		INSERT INTO jobs (pipeline_id, kind, status, payload, max_attempt, updated_at)
		VALUES (?, ?, 'pending', ?, ?, CURRENT_TIMESTAMP)`, pipelineID, kind, blob, maxAttempt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetJob loads a job by id.
func (db *DB) GetJob(id int64) (*Job, error) {
	var j Job
	var parent sql.NullInt64
	var claimedBy sql.NullString
	var claimedAt sql.NullTime
	err := db.conn.QueryRow(`
		SELECT id, pipeline_id, kind, status, payload, attempt, max_attempt, error, parent_job_id,
		       claimed_by, claimed_at, created_at, updated_at
		FROM jobs WHERE id = ?`, id,
	).Scan(&j.ID, &j.PipelineID, &j.Kind, &j.Status, &j.Payload, &j.Attempt, &j.MaxAttempt, &j.Error, &parent,
		&claimedBy, &claimedAt, &j.CreatedAt, &j.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if parent.Valid {
		j.ParentJobID = &parent.Int64
	}
	if claimedBy.Valid {
		j.ClaimedBy = claimedBy.String
	}
	if claimedAt.Valid {
		j.ClaimedAt = claimedAt.Time
	}
	return &j, nil
}

// ClaimNextPendingJob atomically moves the oldest pending job to running and returns it.
// claimedBy should identify this process (e.g. TRACKER_INSTANCE_ID); empty becomes "default".
func (db *DB) ClaimNextPendingJob(claimedBy string) (*Job, error) {
	if claimedBy == "" {
		claimedBy = "default"
	}
	tx, err := db.conn.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var id int64
	err = tx.QueryRow(`SELECT id FROM jobs WHERE status = 'pending' ORDER BY id ASC LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	res, err := tx.Exec(`
		UPDATE jobs SET
			status = 'running',
			attempt = attempt + 1,
			updated_at = CURRENT_TIMESTAMP,
			claimed_by = ?,
			claimed_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'pending'`, claimedBy, id)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		// Lost race: another worker claimed this row.
		return nil, nil
	}

	var j Job
	var parent sql.NullInt64
	var claimedByCol sql.NullString
	var claimedAt sql.NullTime
	err = tx.QueryRow(`
		SELECT id, pipeline_id, kind, status, payload, attempt, max_attempt, error, parent_job_id,
		       claimed_by, claimed_at, created_at, updated_at
		FROM jobs WHERE id = ?`, id,
	).Scan(&j.ID, &j.PipelineID, &j.Kind, &j.Status, &j.Payload, &j.Attempt, &j.MaxAttempt, &j.Error, &parent,
		&claimedByCol, &claimedAt, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if parent.Valid {
		j.ParentJobID = &parent.Int64
	}
	if claimedByCol.Valid {
		j.ClaimedBy = claimedByCol.String
	}
	if claimedAt.Valid {
		j.ClaimedAt = claimedAt.Time
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &j, nil
}

// CompleteJob marks job succeeded.
func (db *DB) CompleteJob(id int64) error {
	_, err := db.conn.Exec(`UPDATE jobs SET status = 'done', error = '', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// FailJob marks failed; if attempt < max_attempt, reset to pending for retry.
func (db *DB) FailJob(id int64, attempt, maxAttempt int, errMsg string) error {
	if attempt < maxAttempt {
		_, err := db.conn.Exec(`UPDATE jobs SET status = 'pending', error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, errMsg, id)
		return err
	}
	_, err := db.conn.Exec(`UPDATE jobs SET status = 'failed', error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, errMsg, id)
	return err
}
