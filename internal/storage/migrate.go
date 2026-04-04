package storage

import "gorm.io/gorm"

func ensureTable(db *gorm.DB, model interface{}) error {
	if db.Migrator().HasTable(model) {
		return nil
	}
	return db.Migrator().CreateTable(model)
}

func ensureColumn(db *gorm.DB, model interface{}, field string) error {
	if db.Migrator().HasColumn(model, field) {
		return nil
	}
	return db.Migrator().AddColumn(model, field)
}

func ensureIndex(db *gorm.DB, model interface{}, name string) error {
	if db.Migrator().HasIndex(model, name) {
		return nil
	}
	return db.Migrator().CreateIndex(model, name)
}

// migrate applies additive schema changes conservatively.
// We avoid full AutoMigrate on legacy SQLite tables because old CREATE TABLE
// definitions that contain FOREIGN KEY clauses can be mis-parsed by sqlite migrator
// and trigger temp-table rebuild failures.
func migrate(db *gorm.DB) error {
	models := []interface{}{
		&Source{},
		&Item{},
		&ormSummary{},
		&Interest{},
		&PipelineDefinition{},
		&ormJob{},
		&PluginPackage{},
		&PluginPackageVersion{},
		&PluginGroup{},
		&PluginGroupVersion{},
	}
	for _, model := range models {
		if err := ensureTable(db, model); err != nil {
			return err
		}
	}

	columns := []struct {
		model interface{}
		field string
	}{
		{&ormJob{}, "ClaimedBy"},
		{&ormJob{}, "ClaimedAt"},
		{&PluginPackage{}, "CurrentVersionID"},
		{&PluginPackage{}, "LastBuildStatus"},
		{&PluginPackage{}, "LastBuildLog"},
		{&PluginPackage{}, "LastBuildAt"},
		{&PluginPackage{}, "LastRunStatus"},
		{&PluginPackage{}, "LastRunLog"},
		{&PluginPackage{}, "LastRunAt"},
		{&PluginGroup{}, "CurrentVersionID"},
		{&PluginGroupVersion{}, "SourcePipelineID"},
		{&PluginGroupVersion{}, "ChangeNote"},
	}
	for _, item := range columns {
		if err := ensureColumn(db, item.model, item.field); err != nil {
			return err
		}
	}

	indexes := []struct {
		model interface{}
		name  string
	}{
		{&ormSummary{}, "idx_summaries_item_id"},
		{&PipelineDefinition{}, "idx_pipeline_definitions_default"},
		{&ormJob{}, "idx_jobs_status"},
		{&ormJob{}, "idx_jobs_pipeline"},
		{&PluginPackageVersion{}, "idx_plugin_package_versions_package"},
		{&PluginGroupVersion{}, "idx_plugin_group_versions_group"},
	}
	for _, item := range indexes {
		if err := ensureIndex(db, item.model, item.name); err != nil {
			return err
		}
	}
	return nil
}
