package storage

import (
	"database/sql"
	"time"
)

// PipelineDefinition is a persisted DAG pipeline (graph_json = PipelineGraph JSON).
type PipelineDefinition struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	GraphJSON string    `json:"graph_json"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PipelineSummary for list API (no full graph).
type PipelineSummary struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"is_default"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListPipelineDefinitions returns all pipelines ordered by id.
func (db *DB) ListPipelineDefinitions() ([]PipelineSummary, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, is_default, updated_at FROM pipeline_definitions ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []PipelineSummary
	for rows.Next() {
		var s PipelineSummary
		var def int
		if err := rows.Scan(&s.ID, &s.Name, &def, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.IsDefault = def != 0
		list = append(list, s)
	}
	return list, rows.Err()
}

// GetPipelineDefinition loads one pipeline by id.
func (db *DB) GetPipelineDefinition(id int64) (*PipelineDefinition, error) {
	var p PipelineDefinition
	var def int
	err := db.conn.QueryRow(`
		SELECT id, name, graph_json, is_default, created_at, updated_at
		FROM pipeline_definitions WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.GraphJSON, &def, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.IsDefault = def != 0
	return &p, nil
}

// GetDefaultPipelineDefinition returns the row marked default, if any.
func (db *DB) GetDefaultPipelineDefinition() (*PipelineDefinition, error) {
	var p PipelineDefinition
	var def int
	err := db.conn.QueryRow(`
		SELECT id, name, graph_json, is_default, created_at, updated_at
		FROM pipeline_definitions WHERE is_default = 1 ORDER BY id LIMIT 1`,
	).Scan(&p.ID, &p.Name, &p.GraphJSON, &def, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.IsDefault = def != 0
	return &p, nil
}

func (db *DB) clearDefaultPipelineFlag(tx *sql.Tx) error {
	_, err := tx.Exec(`UPDATE pipeline_definitions SET is_default = 0, updated_at = CURRENT_TIMESTAMP`)
	return err
}

// CreatePipelineDefinition inserts a pipeline. If isDefault, clears other defaults.
func (db *DB) CreatePipelineDefinition(name, graphJSON string, isDefault bool) (int64, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if isDefault {
		if err := db.clearDefaultPipelineFlag(tx); err != nil {
			return 0, err
		}
	}
	def := 0
	if isDefault {
		def = 1
	}
	res, err := tx.Exec(`
		INSERT INTO pipeline_definitions (name, graph_json, is_default, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)`, name, graphJSON, def)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

// UpdatePipelineDefinition updates name/graph/is_default.
func (db *DB) UpdatePipelineDefinition(id int64, name, graphJSON string, isDefault bool) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if isDefault {
		if err := db.clearDefaultPipelineFlag(tx); err != nil {
			return err
		}
	}
	def := 0
	if isDefault {
		def = 1
	}
	_, err = tx.Exec(`
		UPDATE pipeline_definitions SET name = ?, graph_json = ?, is_default = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, name, graphJSON, def, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// DeletePipelineDefinition removes a pipeline by id.
func (db *DB) DeletePipelineDefinition(id int64) error {
	_, err := db.conn.Exec(`DELETE FROM pipeline_definitions WHERE id = ?`, id)
	return err
}
