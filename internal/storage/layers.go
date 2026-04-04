package storage

import (
	"time"

	"gorm.io/gorm"
)

type SourceRepository interface {
	List() ([]Source, error)
	Create(url, typ, config string) (int64, error)
	Delete(id int64) error
}

type InterestRepository interface {
	List() ([]Interest, error)
	Create(name, keywords, config string) (int64, error)
	Delete(id int64) error
}

type ItemRepository interface {
	Save(sourceID *int64, title, url, content, summary string, ts time.Time, raw string) (int64, error)
	SaveSummary(itemID int64, summary, keyPointsJSON string) (int64, error)
	List(limit, offset int) ([]Item, error)
	Count() (int64, error)
	ListSummaries(limit, offset int) ([]SummaryRow, error)
	CountSummaries() (int64, error)
}

type PipelineRepository interface {
	ListDefinitions() ([]PipelineSummary, error)
	GetDefinition(id int64) (*PipelineDefinition, error)
	GetDefaultDefinition() (*PipelineDefinition, error)
	CreateDefinition(name, graphJSON string, isDefault bool) (int64, error)
	UpdateDefinition(id int64, name, graphJSON string, isDefault bool) error
	DeleteDefinition(id int64) error
}

type JobRepository interface {
	Enqueue(pipelineID int64, kind string, payload map[string]interface{}, maxAttempt int) (int64, error)
	Get(id int64) (*Job, error)
	ClaimNext(claimedBy string) (*Job, error)
	Complete(id int64) error
	Fail(id int64, attempt, maxAttempt int, errMsg string) error
}

type PluginPackageRepository interface {
	List() ([]PluginPackage, error)
	Get(id int64) (*PluginPackage, error)
	GetVersion(id int64) (*PluginPackageVersion, error)
	GetCurrentVersion(packageID int64) (*PluginPackageVersion, error)
	Create(pkg PluginPackage, ver PluginPackageVersion) (int64, int64, error)
	CreateVersion(packageID int64, ver PluginPackageVersion, makeCurrent bool) (int64, error)
	Update(pkg PluginPackage) error
	UpdateVersion(ver PluginPackageVersion) error
}

type PluginGroupRepository interface {
	List() ([]PluginGroup, error)
	Get(id int64) (*PluginGroup, error)
	GetByName(name string) (*PluginGroup, error)
	Create(name, description, graphJSON, ioJSON string) (int64, error)
	Update(id int64, name, description, graphJSON, ioJSON string) error
	Delete(id int64) error
	ListVersions(groupID int64) ([]PluginGroupVersion, error)
	GetVersion(id int64) (*PluginGroupVersion, error)
	CreateVersion(groupID int64, version, graphJSON, ioJSON string, sourcePipelineID *int64, changeNote string, makeCurrent bool) (int64, error)
	CountPipelineRefs(groupID int64) (int, int, error)
}

type Repositories struct {
	Sources        SourceRepository
	Interests      InterestRepository
	Items          ItemRepository
	Pipelines      PipelineRepository
	Jobs           JobRepository
	PluginPackages PluginPackageRepository
	PluginGroups   PluginGroupRepository
}

type PipelineService interface {
	Save(id *int64, name, graphJSON string, isDefault bool) (int64, error)
	GetDefault() (*PipelineDefinition, error)
}

type PluginPlatformService interface {
	CreatePackageRelease(pkg PluginPackage, ver PluginPackageVersion) (int64, int64, error)
	PublishGroup(name, description, graphJSON, ioJSON string) (int64, error)
	PublishGroupVersion(groupID int64, name, description, graphJSON, ioJSON string) error
	UpsertGroup(name, description, graphJSON, ioJSON string) (*PluginGroup, bool, error)
	LoadGroup(groupID int64) (*PluginGroup, error)
}

type Services struct {
	Pipelines      PipelineService
	PluginPlatform PluginPlatformService
}

type sourceRepo struct{ db *DB }
type interestRepo struct{ db *DB }
type itemRepo struct{ db *DB }
type pipelineRepo struct{ db *DB }
type jobRepo struct{ db *DB }
type pluginPackageRepo struct{ db *DB }
type pluginGroupRepo struct{ db *DB }

type pipelineServiceImpl struct{ repos Repositories }
type pluginPlatformServiceImpl struct{ repos Repositories }

func (db *DB) Repositories() Repositories {
	return Repositories{
		Sources:        sourceRepo{db: db},
		Interests:      interestRepo{db: db},
		Items:          itemRepo{db: db},
		Pipelines:      pipelineRepo{db: db},
		Jobs:           jobRepo{db: db},
		PluginPackages: pluginPackageRepo{db: db},
		PluginGroups:   pluginGroupRepo{db: db},
	}
}

func (db *DB) Services() Services {
	repos := db.Repositories()
	return Services{
		Pipelines:      pipelineServiceImpl{repos: repos},
		PluginPlatform: pluginPlatformServiceImpl{repos: repos},
	}
}

func (db *DB) InTx(fn func(tx *DB) error) error {
	return db.orm.Transaction(func(tx *gorm.DB) error {
		return fn(&DB{orm: tx})
	})
}

func (r sourceRepo) List() ([]Source, error) { return r.db.ListSources() }
func (r sourceRepo) Create(url, typ, config string) (int64, error) {
	return r.db.AddSource(url, typ, config)
}
func (r sourceRepo) Delete(id int64) error { return r.db.DeleteSource(id) }

func (r interestRepo) List() ([]Interest, error) { return r.db.ListInterests() }
func (r interestRepo) Create(name, keywords, config string) (int64, error) {
	return r.db.AddInterest(name, keywords, config)
}
func (r interestRepo) Delete(id int64) error { return r.db.DeleteInterest(id) }

func (r itemRepo) Save(sourceID *int64, title, url, content, summary string, ts time.Time, raw string) (int64, error) {
	return r.db.SaveItem(sourceID, title, url, content, summary, ts, raw)
}
func (r itemRepo) SaveSummary(itemID int64, summary, keyPointsJSON string) (int64, error) {
	return r.db.SaveSummary(itemID, summary, keyPointsJSON)
}
func (r itemRepo) List(limit, offset int) ([]Item, error) { return r.db.ListItems(limit, offset) }
func (r itemRepo) Count() (int64, error)                  { return r.db.CountItems() }
func (r itemRepo) ListSummaries(limit, offset int) ([]SummaryRow, error) {
	return r.db.ListSummaries(limit, offset)
}
func (r itemRepo) CountSummaries() (int64, error) { return r.db.CountSummaries() }

func (r pipelineRepo) ListDefinitions() ([]PipelineSummary, error) {
	return r.db.ListPipelineDefinitions()
}
func (r pipelineRepo) GetDefinition(id int64) (*PipelineDefinition, error) {
	return r.db.GetPipelineDefinition(id)
}
func (r pipelineRepo) GetDefaultDefinition() (*PipelineDefinition, error) {
	return r.db.GetDefaultPipelineDefinition()
}
func (r pipelineRepo) CreateDefinition(name, graphJSON string, isDefault bool) (int64, error) {
	return r.db.CreatePipelineDefinition(name, graphJSON, isDefault)
}
func (r pipelineRepo) UpdateDefinition(id int64, name, graphJSON string, isDefault bool) error {
	return r.db.UpdatePipelineDefinition(id, name, graphJSON, isDefault)
}
func (r pipelineRepo) DeleteDefinition(id int64) error { return r.db.DeletePipelineDefinition(id) }

func (r jobRepo) Enqueue(pipelineID int64, kind string, payload map[string]interface{}, maxAttempt int) (int64, error) {
	return r.db.EnqueueJob(pipelineID, kind, payload, maxAttempt)
}
func (r jobRepo) Get(id int64) (*Job, error) { return r.db.GetJob(id) }
func (r jobRepo) ClaimNext(claimedBy string) (*Job, error) {
	return r.db.ClaimNextPendingJob(claimedBy)
}
func (r jobRepo) Complete(id int64) error { return r.db.CompleteJob(id) }
func (r jobRepo) Fail(id int64, attempt, maxAttempt int, errMsg string) error {
	return r.db.FailJob(id, attempt, maxAttempt, errMsg)
}

func (r pluginPackageRepo) List() ([]PluginPackage, error)       { return r.db.ListPluginPackages() }
func (r pluginPackageRepo) Get(id int64) (*PluginPackage, error) { return r.db.GetPluginPackage(id) }
func (r pluginPackageRepo) GetVersion(id int64) (*PluginPackageVersion, error) {
	return r.db.GetPluginPackageVersion(id)
}
func (r pluginPackageRepo) GetCurrentVersion(packageID int64) (*PluginPackageVersion, error) {
	return r.db.GetCurrentPluginPackageVersion(packageID)
}
func (r pluginPackageRepo) Create(pkg PluginPackage, ver PluginPackageVersion) (int64, int64, error) {
	return r.db.CreatePluginPackage(pkg, ver)
}
func (r pluginPackageRepo) CreateVersion(packageID int64, ver PluginPackageVersion, makeCurrent bool) (int64, error) {
	return r.db.CreatePluginPackageVersion(packageID, ver, makeCurrent)
}
func (r pluginPackageRepo) Update(pkg PluginPackage) error { return r.db.UpdatePluginPackage(pkg) }
func (r pluginPackageRepo) UpdateVersion(ver PluginPackageVersion) error {
	return r.db.UpdatePluginPackageVersion(ver)
}

func (r pluginGroupRepo) List() ([]PluginGroup, error)       { return r.db.ListPluginGroups() }
func (r pluginGroupRepo) Get(id int64) (*PluginGroup, error) { return r.db.GetPluginGroup(id) }
func (r pluginGroupRepo) GetByName(name string) (*PluginGroup, error) {
	return r.db.GetPluginGroupByName(name)
}
func (r pluginGroupRepo) Create(name, description, graphJSON, ioJSON string) (int64, error) {
	return r.db.CreatePluginGroup(name, description, graphJSON, ioJSON)
}
func (r pluginGroupRepo) Update(id int64, name, description, graphJSON, ioJSON string) error {
	return r.db.UpdatePluginGroup(id, name, description, graphJSON, ioJSON)
}
func (r pluginGroupRepo) Delete(id int64) error { return r.db.DeletePluginGroup(id) }
func (r pluginGroupRepo) ListVersions(groupID int64) ([]PluginGroupVersion, error) {
	return r.db.ListPluginGroupVersions(groupID)
}
func (r pluginGroupRepo) GetVersion(id int64) (*PluginGroupVersion, error) {
	return r.db.GetPluginGroupVersion(id)
}
func (r pluginGroupRepo) CreateVersion(groupID int64, version, graphJSON, ioJSON string, sourcePipelineID *int64, changeNote string, makeCurrent bool) (int64, error) {
	return r.db.CreatePluginGroupVersion(groupID, version, graphJSON, ioJSON, sourcePipelineID, changeNote, makeCurrent)
}
func (r pluginGroupRepo) CountPipelineRefs(groupID int64) (int, int, error) {
	return r.db.CountPipelinesReferencingPluginGroup(groupID)
}

func (s pipelineServiceImpl) Save(id *int64, name, graphJSON string, isDefault bool) (int64, error) {
	if id == nil || *id == 0 {
		return s.repos.Pipelines.CreateDefinition(name, graphJSON, isDefault)
	}
	return *id, s.repos.Pipelines.UpdateDefinition(*id, name, graphJSON, isDefault)
}

func (s pipelineServiceImpl) GetDefault() (*PipelineDefinition, error) {
	return s.repos.Pipelines.GetDefaultDefinition()
}

func (s pluginPlatformServiceImpl) CreatePackageRelease(pkg PluginPackage, ver PluginPackageVersion) (int64, int64, error) {
	return s.repos.PluginPackages.Create(pkg, ver)
}

func (s pluginPlatformServiceImpl) PublishGroup(name, description, graphJSON, ioJSON string) (int64, error) {
	return s.repos.PluginGroups.Create(name, description, graphJSON, ioJSON)
}

func (s pluginPlatformServiceImpl) PublishGroupVersion(groupID int64, name, description, graphJSON, ioJSON string) error {
	return s.repos.PluginGroups.Update(groupID, name, description, graphJSON, ioJSON)
}

func (s pluginPlatformServiceImpl) UpsertGroup(name, description, graphJSON, ioJSON string) (*PluginGroup, bool, error) {
	existing, err := s.repos.PluginGroups.GetByName(name)
	if err != nil {
		return nil, false, err
	}
	if existing == nil {
		id, err := s.repos.PluginGroups.Create(name, description, graphJSON, ioJSON)
		if err != nil {
			return nil, false, err
		}
		group, err := s.repos.PluginGroups.Get(id)
		return group, true, err
	}
	if err := s.repos.PluginGroups.Update(existing.ID, name, description, graphJSON, ioJSON); err != nil {
		return nil, false, err
	}
	group, err := s.repos.PluginGroups.Get(existing.ID)
	return group, false, err
}

func (s pluginPlatformServiceImpl) LoadGroup(groupID int64) (*PluginGroup, error) {
	return s.repos.PluginGroups.Get(groupID)
}
