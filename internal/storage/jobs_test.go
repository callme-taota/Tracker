package storage

import "testing"

func TestClaimNextPendingJob_serial(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/t.db"
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.EnqueueJob(1, "run", nil, 3)
	if err != nil {
		t.Fatal(err)
	}
	j, err := db.ClaimNextPendingJob("worker-a")
	if err != nil || j == nil {
		t.Fatalf("claim: %v %#v", err, j)
	}
	if j.Status != "running" || j.ClaimedBy != "worker-a" {
		t.Fatalf("bad job: %+v", j)
	}
	j2, err := db.ClaimNextPendingJob("worker-b")
	if err != nil {
		t.Fatal(err)
	}
	if j2 != nil {
		t.Fatalf("expected no second job, got %+v", j2)
	}
}
