package storage

import (
	"time"

	"gorm.io/gorm"
)

// PipelineDefinition is a persisted DAG pipeline (graph_json = PipelineGraph JSON).
type PipelineDefinition struct {
	ID        int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"column:name;not null"`
	GraphJSON string    `json:"graph_json" gorm:"column:graph_json;not null"`
	IsDefault bool      `json:"is_default" gorm:"column:is_default;not null;default:false;index:idx_pipeline_definitions_default"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoCreateTime;autoUpdateTime"`
}

func (PipelineDefinition) TableName() string { return "pipeline_definitions" }

// PipelineSummary for list API (no full graph).
type PipelineSummary struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"is_default"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListPipelineDefinitions returns all pipelines ordered by id.
func (db *DB) ListPipelineDefinitions() ([]PipelineSummary, error) {
	var list []PipelineSummary
	err := db.orm.Model(&PipelineDefinition{}).
		Select("id, name, is_default, updated_at").
		Order("id").
		Scan(&list).Error
	return list, err
}

// GetPipelineDefinition loads one pipeline by id.
func (db *DB) GetPipelineDefinition(id int64) (*PipelineDefinition, error) {
	var row PipelineDefinition
	err := db.orm.First(&row, id).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// GetDefaultPipelineDefinition returns the row marked default, if any.
func (db *DB) GetDefaultPipelineDefinition() (*PipelineDefinition, error) {
	var row PipelineDefinition
	err := db.orm.Where("is_default = ?", true).Order("id").First(&row).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func clearDefaultPipelineFlag(tx *DB) error {
	return tx.orm.Model(&PipelineDefinition{}).
		Where("is_default = ?", true).
		Update("is_default", false).Error
}

// CreatePipelineDefinition inserts a pipeline. If isDefault, clears other defaults.
func (db *DB) CreatePipelineDefinition(name, graphJSON string, isDefault bool) (int64, error) {
	var created PipelineDefinition
	err := db.orm.Transaction(func(tx *gorm.DB) error {
		w := &DB{orm: tx}
		if isDefault {
			if err := clearDefaultPipelineFlag(w); err != nil {
				return err
			}
		}
		created = PipelineDefinition{Name: name, GraphJSON: graphJSON, IsDefault: isDefault}
		return tx.Create(&created).Error
	})
	if err != nil {
		return 0, err
	}
	return created.ID, nil
}

// UpdatePipelineDefinition updates name/graph/is_default.
func (db *DB) UpdatePipelineDefinition(id int64, name, graphJSON string, isDefault bool) error {
	return db.orm.Transaction(func(tx *gorm.DB) error {
		w := &DB{orm: tx}
		if isDefault {
			if err := clearDefaultPipelineFlag(w); err != nil {
				return err
			}
		}
		return tx.Model(&PipelineDefinition{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"name":       name,
				"graph_json": graphJSON,
				"is_default": isDefault,
			}).Error
	})
}

// DeletePipelineDefinition removes a pipeline by id.
func (db *DB) DeletePipelineDefinition(id int64) error {
	return db.orm.Delete(&PipelineDefinition{}, id).Error
}
