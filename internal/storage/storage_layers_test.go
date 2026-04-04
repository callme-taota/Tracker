package storage

import (
	"fmt"
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRepositoriesAndServicesExposeLayers(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	repos := db.Repositories()
	svcs := db.Services()

	if repos.Sources == nil || repos.Pipelines == nil || repos.PluginGroups == nil {
		t.Fatal("repositories should be initialized")
	}
	if svcs.Pipelines == nil || svcs.PluginPlatform == nil {
		t.Fatal("services should be initialized")
	}
}

func TestPipelineServiceSaveSwitchesDefault(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	svc := db.Services().Pipelines
	id1, err := svc.Save(nil, "p1", `{"name":"p1","nodes":[],"edges":[]}`, true)
	if err != nil {
		t.Fatal(err)
	}
	id2, err := svc.Save(nil, "p2", `{"name":"p2","nodes":[],"edges":[]}`, true)
	if err != nil {
		t.Fatal(err)
	}
	if id1 == id2 {
		t.Fatal("expected different pipeline ids")
	}

	def, err := svc.GetDefault()
	if err != nil {
		t.Fatal(err)
	}
	if def == nil || def.ID != id2 {
		t.Fatalf("want pipeline %d as default, got %+v", id2, def)
	}

	first, err := db.GetPipelineDefinition(id1)
	if err != nil {
		t.Fatal(err)
	}
	if first == nil || first.IsDefault {
		t.Fatalf("first pipeline should no longer be default: %+v", first)
	}
}

func TestJobRepositoryRetryLifecycle(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	repo := db.Repositories().Jobs
	jobID, err := repo.Enqueue(11, "run", map[string]interface{}{"from": "today"}, 3)
	if err != nil {
		t.Fatal(err)
	}

	job, err := repo.ClaimNext("worker-a")
	if err != nil {
		t.Fatal(err)
	}
	if job == nil || job.ID != jobID || job.Status != "running" || job.Attempt != 1 {
		t.Fatalf("unexpected claimed job: %+v", job)
	}

	if err := repo.Fail(job.ID, job.Attempt, job.MaxAttempt, "boom"); err != nil {
		t.Fatal(err)
	}
	reloaded, err := repo.Get(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded == nil || reloaded.Status != "pending" || reloaded.Error != "boom" {
		t.Fatalf("job should return to pending after retryable failure: %+v", reloaded)
	}

	job2, err := repo.ClaimNext("worker-b")
	if err != nil {
		t.Fatal(err)
	}
	if job2 == nil || job2.Attempt != 2 || job2.ClaimedBy != "worker-b" {
		t.Fatalf("unexpected retried claim: %+v", job2)
	}

	if err := repo.Complete(job2.ID); err != nil {
		t.Fatal(err)
	}
	done, err := repo.Get(job2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done == nil || done.Status != "done" || done.Error != "" {
		t.Fatalf("completed job should be done with cleared error: %+v", done)
	}
}

func TestPluginGroupVersioningAndReferenceCounts(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	svc := db.Services().PluginPlatform
	groupID, err := svc.PublishGroup("group-a", "desc", `{"name":"g","nodes":[],"edges":[]}`, ``)
	if err != nil {
		t.Fatal(err)
	}

	group, err := svc.LoadGroup(groupID)
	if err != nil {
		t.Fatal(err)
	}
	if group == nil || group.CurrentVersion != "v1" || group.VersionCount != 1 {
		t.Fatalf("unexpected initial group aggregate: %+v", group)
	}

	if err := svc.PublishGroupVersion(groupID, "group-a", "desc-2", `{"name":"g2","nodes":[],"edges":[]}`, ``); err != nil {
		t.Fatal(err)
	}
	group, err = svc.LoadGroup(groupID)
	if err != nil {
		t.Fatal(err)
	}
	if group == nil || group.CurrentVersion != "v2" || group.VersionCount != 2 {
		t.Fatalf("unexpected updated group aggregate: %+v", group)
	}

	versions, err := db.ListPluginGroupVersions(groupID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("want 2 versions, got %d", len(versions))
	}
	oldVersionID := versions[1].ID
	graphJSON := `{"name":"pipe","nodes":[],"edges":[],"group_refs":[{"group_id":` +
		int64ToString(groupID) + `,"group_version_id":` + int64ToString(oldVersionID) + `}]}`
	if _, err := db.CreatePipelineDefinition("pipe-with-old-ref", graphJSON, false); err != nil {
		t.Fatal(err)
	}

	refs, outdated, err := db.CountPipelinesReferencingPluginGroup(groupID)
	if err != nil {
		t.Fatal(err)
	}
	if refs != 1 || outdated != 1 {
		t.Fatalf("want refs=1 outdated=1 got refs=%d outdated=%d", refs, outdated)
	}
}

func TestPluginPackageServiceCreatesCurrentVersion(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	svc := db.Services().PluginPlatform
	pkgID, verID, err := svc.CreatePackageRelease(
		PluginPackage{
			PluginID:     "demo",
			Name:         "Demo",
			Runtime:      "go",
			SourceKind:   "workspace",
			ReviewStatus: "approved",
			RiskLevel:    "low",
		},
		PluginPackageVersion{
			Version:      "0.1.0",
			ManifestJSON: `{}`,
			EntryFile:    "main.go",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if pkgID <= 0 || verID <= 0 {
		t.Fatalf("invalid ids: pkg=%d ver=%d", pkgID, verID)
	}

	current, err := db.GetCurrentPluginPackageVersion(pkgID)
	if err != nil {
		t.Fatal(err)
	}
	if current == nil || current.ID != verID {
		t.Fatalf("want current version %d got %+v", verID, current)
	}
}

func TestPluginPlatformServiceUpsertGroupByNamePublishesVersion(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	svc := db.Services().PluginPlatform
	group, created, err := svc.UpsertGroup("same-name", "v1", `{"name":"g1","nodes":[],"edges":[]}`, "")
	if err != nil {
		t.Fatal(err)
	}
	if !created || group == nil || group.CurrentVersion != "v1" {
		t.Fatalf("expected initial create, got created=%v group=%+v", created, group)
	}

	group, created, err = svc.UpsertGroup("same-name", "v2", `{"name":"g2","nodes":[],"edges":[]}`, "")
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("expected second upsert to update existing group")
	}
	if group == nil || group.CurrentVersion != "v2" || group.VersionCount != 2 {
		t.Fatalf("expected published version on existing group, got %+v", group)
	}
}

func int64ToString(v int64) string {
	return fmt.Sprintf("%d", v)
}
