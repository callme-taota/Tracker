package storage

import (
	"strconv"
	"time"

	"gorm.io/gorm"
)

type PluginPackage struct {
	ID               int64      `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PluginID         string     `json:"plugin_id" gorm:"column:plugin_id;not null;uniqueIndex"`
	Name             string     `json:"name" gorm:"column:name;not null"`
	Runtime          string     `json:"runtime" gorm:"column:runtime;not null"`
	SourceKind       string     `json:"source_kind" gorm:"column:source_kind;not null"`
	ReviewStatus     string     `json:"review_status" gorm:"column:review_status;not null;default:pending"`
	RiskLevel        string     `json:"risk_level" gorm:"column:risk_level;not null;default:unknown"`
	Enabled          bool       `json:"enabled" gorm:"column:enabled;not null;default:false"`
	CurrentVersionID *int64     `json:"current_version_id,omitempty" gorm:"column:current_version_id"`
	LastBuildStatus  string     `json:"last_build_status,omitempty" gorm:"column:last_build_status"`
	LastBuildLog     string     `json:"last_build_log,omitempty" gorm:"column:last_build_log"`
	LastBuildAt      *time.Time `json:"last_build_at,omitempty" gorm:"column:last_build_at"`
	LastRunStatus    string     `json:"last_run_status,omitempty" gorm:"column:last_run_status"`
	LastRunLog       string     `json:"last_run_log,omitempty" gorm:"column:last_run_log"`
	LastRunAt        *time.Time `json:"last_run_at,omitempty" gorm:"column:last_run_at"`
	CreatedAt        time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time  `json:"updated_at" gorm:"column:updated_at;autoCreateTime;autoUpdateTime"`
}

func (PluginPackage) TableName() string { return "plugin_packages" }

type PluginPackageVersion struct {
	ID               int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PackageID        int64     `json:"package_id" gorm:"column:package_id;not null;index:idx_plugin_package_versions_package"`
	Version          string    `json:"version" gorm:"column:version;not null"`
	ManifestJSON     string    `json:"manifest_json" gorm:"column:manifest_json;not null"`
	EntryFile        string    `json:"entry_file" gorm:"column:entry_file;not null"`
	BuildCommand     string    `json:"build_command" gorm:"column:build_command"`
	RunCommand       string    `json:"run_command" gorm:"column:run_command"`
	ReviewReportJSON string    `json:"review_report_json" gorm:"column:review_report_json"`
	SourceChecksum   string    `json:"source_checksum" gorm:"column:source_checksum"`
	CodeDir          string    `json:"code_dir" gorm:"column:code_dir"`
	CreatedAt        time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (PluginPackageVersion) TableName() string { return "plugin_package_versions" }

type PluginGroup struct {
	ID               int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name             string    `json:"name" gorm:"column:name;not null;uniqueIndex"`
	Description      string    `json:"description" gorm:"column:description"`
	GraphJSON        string    `json:"graph_json" gorm:"column:graph_json;not null"`
	IOJSON           string    `json:"io_json" gorm:"column:io_json"`
	CurrentVersionID *int64    `json:"current_version_id,omitempty" gorm:"column:current_version_id"`
	CurrentVersion   string    `json:"current_version,omitempty" gorm:"-"`
	VersionCount     int       `json:"version_count" gorm:"-"`
	ReferenceCount   int       `json:"reference_count" gorm:"-"`
	OutdatedRefCount int       `json:"outdated_ref_count" gorm:"-"`
	CreatedAt        time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"column:updated_at;autoCreateTime;autoUpdateTime"`
}

func (PluginGroup) TableName() string { return "plugin_groups" }

type PluginGroupVersion struct {
	ID               int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	GroupID          int64     `json:"group_id" gorm:"column:group_id;not null;index:idx_plugin_group_versions_group"`
	Version          string    `json:"version" gorm:"column:version;not null"`
	GraphJSON        string    `json:"graph_json" gorm:"column:graph_json;not null"`
	IOJSON           string    `json:"io_json" gorm:"column:io_json"`
	SourcePipelineID *int64    `json:"source_pipeline_id,omitempty" gorm:"column:source_pipeline_id"`
	ChangeNote       string    `json:"change_note" gorm:"column:change_note"`
	CreatedAt        time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (PluginGroupVersion) TableName() string { return "plugin_group_versions" }

func (db *DB) ListPluginPackages() ([]PluginPackage, error) {
	var list []PluginPackage
	err := db.orm.Order("updated_at DESC").Order("id DESC").Find(&list).Error
	return list, err
}

func (db *DB) GetPluginPackage(id int64) (*PluginPackage, error) {
	var row PluginPackage
	err := db.orm.First(&row, id).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (db *DB) CreatePluginPackage(pkg PluginPackage, ver PluginPackageVersion) (int64, int64, error) {
	var pkgID, verID int64
	err := db.orm.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&pkg).Error; err != nil {
			return err
		}
		ver.PackageID = pkg.ID
		if err := tx.Create(&ver).Error; err != nil {
			return err
		}
		pkg.CurrentVersionID = &ver.ID
		if err := tx.Model(&PluginPackage{}).Where("id = ?", pkg.ID).Update("current_version_id", ver.ID).Error; err != nil {
			return err
		}
		pkgID, verID = pkg.ID, ver.ID
		return nil
	})
	return pkgID, verID, err
}

func (db *DB) UpdatePluginPackage(pkg PluginPackage) error {
	return db.orm.Model(&PluginPackage{}).Where("id = ?", pkg.ID).Updates(map[string]interface{}{
		"plugin_id":          pkg.PluginID,
		"name":               pkg.Name,
		"runtime":            pkg.Runtime,
		"source_kind":        pkg.SourceKind,
		"review_status":      pkg.ReviewStatus,
		"risk_level":         pkg.RiskLevel,
		"enabled":            pkg.Enabled,
		"current_version_id": pkg.CurrentVersionID,
		"last_build_status":  pkg.LastBuildStatus,
		"last_build_log":     pkg.LastBuildLog,
		"last_build_at":      pkg.LastBuildAt,
		"last_run_status":    pkg.LastRunStatus,
		"last_run_log":       pkg.LastRunLog,
		"last_run_at":        pkg.LastRunAt,
	}).Error
}

func (db *DB) GetPluginPackageVersion(id int64) (*PluginPackageVersion, error) {
	var row PluginPackageVersion
	err := db.orm.First(&row, id).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (db *DB) GetCurrentPluginPackageVersion(packageID int64) (*PluginPackageVersion, error) {
	pkg, err := db.GetPluginPackage(packageID)
	if err != nil || pkg == nil || pkg.CurrentVersionID == nil {
		return nil, err
	}
	return db.GetPluginPackageVersion(*pkg.CurrentVersionID)
}

func (db *DB) CreatePluginPackageVersion(packageID int64, ver PluginPackageVersion, makeCurrent bool) (int64, error) {
	var id int64
	err := db.orm.Transaction(func(tx *gorm.DB) error {
		ver.PackageID = packageID
		if err := tx.Create(&ver).Error; err != nil {
			return err
		}
		if makeCurrent {
			if err := tx.Model(&PluginPackage{}).Where("id = ?", packageID).Update("current_version_id", ver.ID).Error; err != nil {
				return err
			}
		}
		id = ver.ID
		return nil
	})
	return id, err
}

func (db *DB) UpdatePluginPackageVersion(ver PluginPackageVersion) error {
	return db.orm.Model(&PluginPackageVersion{}).Where("id = ?", ver.ID).Updates(map[string]interface{}{
		"version":            ver.Version,
		"manifest_json":      ver.ManifestJSON,
		"entry_file":         ver.EntryFile,
		"build_command":      ver.BuildCommand,
		"run_command":        ver.RunCommand,
		"review_report_json": ver.ReviewReportJSON,
		"source_checksum":    ver.SourceChecksum,
		"code_dir":           ver.CodeDir,
	}).Error
}

func hydratePluginGroup(db *gorm.DB, row *PluginGroup) error {
	var count int64
	if err := db.Model(&PluginGroupVersion{}).Where("group_id = ?", row.ID).Count(&count).Error; err != nil {
		return err
	}
	row.VersionCount = int(count)
	if row.CurrentVersionID != nil {
		var ver PluginGroupVersion
		if err := db.Select("id, version").First(&ver, *row.CurrentVersionID).Error; err == nil {
			row.CurrentVersion = ver.Version
		} else if !isNotFound(err) {
			return err
		}
	}
	return nil
}

func (db *DB) ListPluginGroups() ([]PluginGroup, error) {
	var list []PluginGroup
	if err := db.orm.Order("updated_at DESC").Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		if err := hydratePluginGroup(db.orm, &list[i]); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (db *DB) GetPluginGroup(id int64) (*PluginGroup, error) {
	var row PluginGroup
	err := db.orm.First(&row, id).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if err := hydratePluginGroup(db.orm, &row); err != nil {
		return nil, err
	}
	return &row, nil
}

func (db *DB) GetPluginGroupByName(name string) (*PluginGroup, error) {
	var row PluginGroup
	err := db.orm.Where("name = ?", name).First(&row).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if err := hydratePluginGroup(db.orm, &row); err != nil {
		return nil, err
	}
	return &row, nil
}

func (db *DB) CreatePluginGroup(name, description, graphJSON, ioJSON string) (int64, error) {
	var groupID int64
	err := db.orm.Transaction(func(tx *gorm.DB) error {
		group := PluginGroup{Name: name, Description: description, GraphJSON: graphJSON, IOJSON: ioJSON}
		if err := tx.Create(&group).Error; err != nil {
			return err
		}
		ver := PluginGroupVersion{GroupID: group.ID, Version: "v1", GraphJSON: graphJSON, IOJSON: ioJSON}
		if err := tx.Create(&ver).Error; err != nil {
			return err
		}
		if err := tx.Model(&PluginGroup{}).Where("id = ?", group.ID).Update("current_version_id", ver.ID).Error; err != nil {
			return err
		}
		groupID = group.ID
		return nil
	})
	return groupID, err
}

func (db *DB) UpdatePluginGroup(id int64, name, description, graphJSON, ioJSON string) error {
	return db.orm.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&PluginGroupVersion{}).Where("group_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		ver := PluginGroupVersion{
			GroupID:   id,
			Version:   "v" + strconv.FormatInt(count+1, 10),
			GraphJSON: graphJSON,
			IOJSON:    ioJSON,
		}
		if err := tx.Create(&ver).Error; err != nil {
			return err
		}
		return tx.Model(&PluginGroup{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name":               name,
			"description":        description,
			"graph_json":         graphJSON,
			"io_json":            ioJSON,
			"current_version_id": ver.ID,
		}).Error
	})
}

func (db *DB) DeletePluginGroup(id int64) error {
	return db.orm.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", id).Delete(&PluginGroupVersion{}).Error; err != nil {
			return err
		}
		return tx.Delete(&PluginGroup{}, id).Error
	})
}

func (db *DB) ListPluginGroupVersions(groupID int64) ([]PluginGroupVersion, error) {
	var list []PluginGroupVersion
	err := db.orm.Where("group_id = ?", groupID).Order("id DESC").Find(&list).Error
	return list, err
}

func (db *DB) CreatePluginGroupVersion(groupID int64, version, graphJSON, ioJSON string, sourcePipelineID *int64, changeNote string, makeCurrent bool) (int64, error) {
	var id int64
	err := db.orm.Transaction(func(tx *gorm.DB) error {
		row := PluginGroupVersion{
			GroupID:          groupID,
			Version:          version,
			GraphJSON:        graphJSON,
			IOJSON:           ioJSON,
			SourcePipelineID: sourcePipelineID,
			ChangeNote:       changeNote,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if makeCurrent {
			if err := tx.Model(&PluginGroup{}).Where("id = ?", groupID).Updates(map[string]interface{}{
				"graph_json":         graphJSON,
				"io_json":            ioJSON,
				"current_version_id": row.ID,
			}).Error; err != nil {
				return err
			}
		}
		id = row.ID
		return nil
	})
	return id, err
}

func (db *DB) GetPluginGroupVersion(id int64) (*PluginGroupVersion, error) {
	var row PluginGroupVersion
	err := db.orm.First(&row, id).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (db *DB) CountPipelinesReferencingPluginGroup(groupID int64) (int, int, error) {
	var defs []PipelineDefinition
	if err := db.orm.Select("graph_json").Find(&defs).Error; err != nil {
		return 0, 0, err
	}
	current, err := db.GetPluginGroup(groupID)
	if err != nil {
		return 0, 0, err
	}
	var currentVersionID *int64
	if current != nil {
		currentVersionID = current.CurrentVersionID
	}
	refs, outdated := countGroupRefsFromGraphs(defs, groupID, currentVersionID)
	return refs, outdated, nil
}
