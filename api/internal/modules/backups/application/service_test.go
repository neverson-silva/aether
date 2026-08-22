package application

import (
	"context"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/backups/domain"
	databasedomain "aether/internal/modules/databases/domain"
	"aether/internal/platform/druntime/locks"
	"aether/internal/platform/druntime/queue"
	"aether/internal/platform/storage"
)

func TestMain(m *testing.M) {
	RegisterBackupAdapter(fakeBackupAdapter{})
	os.Exit(m.Run())
}

type fakeBackupAdapter struct{}

func (fakeBackupAdapter) Engine() BackupEngine                             { return EnginePostgres }
func (fakeBackupAdapter) Format() string                                   { return "dump" }
func (fakeBackupAdapter) ContentType() string                              { return "application/octet-stream" }
func (fakeBackupAdapter) Validate(_ context.Context, _ DBDescriptor) error { return nil }
func (fakeBackupAdapter) CreateBackup(_ context.Context, _ DBDescriptor, dest io.Writer) error {
	_, err := dest.Write(make([]byte, 64))
	return err
}
func (fakeBackupAdapter) Restore(_ context.Context, _ DBDescriptor, src io.Reader) error {
	_, err := io.Copy(io.Discard, src)
	return err
}

type fakeStore struct {
	mu       sync.Mutex
	config   *domain.BackupConfiguration
	jobs     map[uuid.UUID]*domain.BackupJob
	restores map[uuid.UUID]*domain.RestoreJob
	nextRun  *time.Time
}

func newFakeStore() *fakeStore {
	return &fakeStore{jobs: map[uuid.UUID]*domain.BackupJob{}, restores: map[uuid.UUID]*domain.RestoreJob{}}
}

func (s *fakeStore) GetConfiguration(_ context.Context, _ uuid.UUID) (*domain.BackupConfiguration, error) {
	return s.config, nil
}
func (s *fakeStore) ListConfigurationsByDatabase(_ context.Context, dbID uuid.UUID) ([]domain.BackupConfiguration, error) {
	if s.config == nil || s.config.DatabaseID != dbID {
		return nil, nil
	}
	return []domain.BackupConfiguration{*s.config}, nil
}
func (s *fakeStore) CreateConfiguration(_ context.Context, cfg *domain.BackupConfiguration) (*domain.BackupConfiguration, error) {
	s.config = cfg
	return cfg, nil
}
func (s *fakeStore) UpdateConfiguration(_ context.Context, cfg *domain.BackupConfiguration) (*domain.BackupConfiguration, error) {
	s.config = cfg
	return cfg, nil
}
func (s *fakeStore) DeleteConfiguration(_ context.Context, _ uuid.UUID) error {
	s.config = nil
	return nil
}
func (s *fakeStore) ListEnabledConfigurations(_ context.Context) ([]domain.BackupConfiguration, error) {
	if s.config != nil && s.config.Enabled {
		return []domain.BackupConfiguration{*s.config}, nil
	}
	return nil, nil
}
func (s *fakeStore) SetConfigurationNextRun(_ context.Context, _ uuid.UUID, next *time.Time) error {
	s.nextRun = next
	return nil
}
func (s *fakeStore) CreateJob(_ context.Context, job *domain.BackupJob) (*domain.BackupJob, error) {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	s.jobs[job.ID] = job
	return job, nil
}
func (s *fakeStore) GetJob(_ context.Context, id uuid.UUID) (*domain.BackupJob, error) {
	j, ok := s.jobs[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return j, nil
}
func (s *fakeStore) UpdateJob(_ context.Context, job *domain.BackupJob) (*domain.BackupJob, error) {
	s.jobs[job.ID] = job
	return job, nil
}
func (s *fakeStore) ListJobsByDatabase(_ context.Context, dbID uuid.UUID, _ int) ([]domain.BackupJob, error) {
	var out []domain.BackupJob
	for _, j := range s.jobs {
		if j.DatabaseID == dbID {
			out = append(out, *j)
		}
	}
	return out, nil
}
func (s *fakeStore) ListActiveJobsByDatabase(_ context.Context, dbID uuid.UUID) ([]domain.BackupJob, error) {
	var out []domain.BackupJob
	for _, j := range s.jobs {
		if j.DatabaseID == dbID && !j.Terminal() {
			out = append(out, *j)
		}
	}
	return out, nil
}
func (s *fakeStore) ListQueuedJobs(_ context.Context, _ int) ([]domain.BackupJob, error) {
	return nil, nil
}
func (s *fakeStore) CreateRestoreJob(_ context.Context, job *domain.RestoreJob) (*domain.RestoreJob, error) {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	s.restores[job.ID] = job
	return job, nil
}
func (s *fakeStore) GetRestoreJob(_ context.Context, id uuid.UUID) (*domain.RestoreJob, error) {
	r, ok := s.restores[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return r, nil
}
func (s *fakeStore) UpdateRestoreJob(_ context.Context, job *domain.RestoreJob) (*domain.RestoreJob, error) {
	s.restores[job.ID] = job
	return job, nil
}
func (s *fakeStore) ListRestoreJobsByTarget(_ context.Context, _ uuid.UUID, _ int) ([]domain.RestoreJob, error) {
	return nil, nil
}
func (s *fakeStore) ListQueuedRestoreJobs(_ context.Context, _ int) ([]domain.RestoreJob, error) {
	return nil, nil
}

type fakeDatabases struct{}

func (fakeDatabases) Get(_ context.Context, id uuid.UUID, _ uuid.UUID) (*databasedomain.Database, error) {
	return &databasedomain.Database{ID: id, Engine: databasedomain.EnginePostgres, ContainerID: "cid", User: "u", DBName: "d", Version: "16"}, nil
}

type fakeCipher struct{}

func (fakeCipher) Decrypt(_ string) (string, error) { return "pass", nil }

type fakeProvider struct {
	mu      sync.Mutex
	objects map[string]int64
}

func newFakeProvider() *fakeProvider { return &fakeProvider{objects: map[string]int64{}} }

func (p *fakeProvider) Capabilities() storage.Capabilities {
	return storage.Capabilities{Streaming: true}
}
func (p *fakeProvider) PutObject(_ context.Context, in storage.PutObjectInput) (*storage.PutObjectOutput, error) {
	n, _ := io.Copy(io.Discard, in.Body)
	p.mu.Lock()
	p.objects[in.Key] = n
	p.mu.Unlock()
	return &storage.PutObjectOutput{Key: in.Key, Size: n}, nil
}
func (p *fakeProvider) GetObject(_ context.Context, in storage.GetObjectInput) (*storage.GetObjectOutput, error) {
	p.mu.Lock()
	size, ok := p.objects[in.Key]
	p.mu.Unlock()
	if !ok {
		return nil, storage.ErrObjectNotFound
	}
	return &storage.GetObjectOutput{Key: in.Key, Body: io.NopCloser(newBytesReader(size)), ContentLength: size}, nil
}
func (p *fakeProvider) HeadObject(_ context.Context, in storage.HeadObjectInput) (*storage.HeadObjectOutput, error) {
	p.mu.Lock()
	size, ok := p.objects[in.Key]
	p.mu.Unlock()
	if !ok {
		return nil, storage.ErrObjectNotFound
	}
	return &storage.HeadObjectOutput{Key: in.Key, ContentLength: size}, nil
}
func (p *fakeProvider) DeleteObject(_ context.Context, in storage.DeleteObjectInput) error {
	p.mu.Lock()
	delete(p.objects, in.Key)
	p.mu.Unlock()
	return nil
}
func (p *fakeProvider) ListObjects(_ context.Context, _ storage.ListObjectsInput) (*storage.ListObjectsOutput, error) {
	return &storage.ListObjectsOutput{}, nil
}
func (p *fakeProvider) CopyObject(_ context.Context, _ storage.CopyObjectInput) (*storage.CopyObjectOutput, error) {
	return &storage.CopyObjectOutput{}, nil
}

type bytesReader struct {
	data []byte
}

func newBytesReader(size int64) *bytesReader {
	return &bytesReader{data: make([]byte, size)}
}
func (r *bytesReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

type fakeDestinations struct{ provider *fakeProvider }

func (d fakeDestinations) GetProvider(_ context.Context, _ uuid.UUID, _ uuid.UUID) (storage.Provider, error) {
	return d.provider, nil
}

type fakeAudit struct{}

func (fakeAudit) Record(_ context.Context, _ uuid.UUID, action, rt, rid, details string) {}

type fakeExecutor struct{}

func (fakeExecutor) Exec(_ context.Context, _ string, _ []string, _ ...string) (string, string, error) {
	return "", "", nil
}
func (fakeExecutor) ExecStream(_ context.Context, _ string, _ []string, stdout io.Writer, _ io.Writer, _ ...string) error {
	_, _ = stdout.Write(make([]byte, 64))
	return nil
}
func (fakeExecutor) ExecIn(_ context.Context, _ string, _ []string, _ io.Reader, _ io.Writer, _ io.Writer, _ ...string) error {
	return nil
}

func (fakeExecutor) ExecAs(_ context.Context, _ string, _ string, _ []string, _ io.Reader, _ ...string) (string, string, error) {
	return "", "", nil
}

type memLock struct{ held bool }

func (m *memLock) Acquire(_ context.Context, _ string, _ time.Duration) (locks.Lock, bool, error) {
	if m.held {
		return locks.Lock{}, false, nil
	}
	m.held = true
	return locks.Lock{Name: "x", Token: "1", TTL: time.Minute}, true, nil
}
func (m *memLock) Renew(_ context.Context, _ locks.Lock, _ time.Duration) error { return nil }
func (m *memLock) Release(_ context.Context, l locks.Lock) error {
	m.held = false
	return nil
}
func (m *memLock) Locked(_ context.Context, _ string) (bool, error) { return m.held, nil }

type memQueue struct{ jobs []queue.Job }

func (q *memQueue) Enqueue(_ context.Context, _ string, job queue.Job) error {
	q.jobs = append(q.jobs, job)
	return nil
}
func (q *memQueue) NewConsumer(_ context.Context, _, _, _ string) (queue.Consumer, error) {
	return nil, nil
}
func (q *memQueue) Len(_ context.Context, _ string) (int64, error)           { return 0, nil }
func (q *memQueue) Pending(_ context.Context, _, _ string) (int64, error)    { return 0, nil }
func (q *memQueue) DeadLetterLen(_ context.Context, _ string) (int64, error) { return 0, nil }
func (q *memQueue) Cancel(_ context.Context, _, _ string) error              { return nil }

func newService() (*DatabaseBackups, *fakeStore, *fakeProvider, *memQueue, *memLock) {
	store := newFakeStore()
	prov := newFakeProvider()
	q := &memQueue{}
	lk := &memLock{}
	return &DatabaseBackups{
		Store: store, Databases: fakeDatabases{}, Passwords: fakeCipher{},
		Destinations: fakeDestinations{provider: prov}, Exec: fakeExecutor{},
		Queue: q, Locks: lk, Audit: fakeAudit{}, Timeout: time.Minute,
	}, store, prov, q, lk
}

func TestUpsertConfigurationComputesNextRun(t *testing.T) {
	svc, store, _, _, _ := newService()
	cfg := &domain.BackupConfiguration{
		DatabaseID: uuid.New(), DestinationID: uuid.New(), Enabled: true,
		PathPrefix: "databases/prod",
		Schedule:   domain.Schedule{Type: domain.ScheduleDaily, At: "03:00", Timezone: "America/Sao_Paulo"},
		Retention:  domain.Retention{Type: domain.RetentionLatest},
	}
	saved, err := svc.UpsertConfiguration(context.Background(), uuid.New(), cfg)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if saved.NextRunAt == nil {
		t.Fatal("next_run_at should be computed")
	}
	if store.nextRun == nil && saved.NextRunAt == nil {
		t.Fatal("next run not set")
	}
}

func TestStartManualBackupWithoutConfigFails(t *testing.T) {
	svc, _, _, q, _ := newService()
	_, err := svc.StartManualBackup(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error when no config")
	}
	if len(q.jobs) != 0 {
		t.Fatal("should not enqueue without config")
	}
}

func TestStartManualBackupEnqueuesAndConflict(t *testing.T) {
	svc, store, _, q, _ := newService()
	dbID := uuid.New()
	store.config = &domain.BackupConfiguration{
		ID: uuid.New(), DatabaseID: dbID, DestinationID: uuid.New(), Enabled: true,
		PathPrefix: "databases/prod",
		Schedule:   domain.Schedule{Type: domain.ScheduleDaily, At: "03:00", Timezone: "UTC"},
		Retention:  domain.Retention{Type: domain.RetentionAll},
	}
	job, err := svc.StartManualBackup(context.Background(), dbID, uuid.New())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if job.Status != domain.BackupQueued {
		t.Fatalf("expected queued, got %s", job.Status)
	}
	if len(q.jobs) != 1 {
		t.Fatalf("expected 1 enqueue, got %d", len(q.jobs))
	}
	if _, err := svc.StartManualBackup(context.Background(), dbID, uuid.New()); err != domain.ErrConflict {
		t.Fatalf("expected conflict for concurrent backup, got %v", err)
	}
}

func TestManualBackupPipelineToCompleted(t *testing.T) {
	svc, store, prov, _, _ := newService()
	dbID := uuid.New()
	destID := uuid.New()
	store.config = &domain.BackupConfiguration{
		ID: uuid.New(), DatabaseID: dbID, DestinationID: destID, Enabled: true,
		PathPrefix: "databases/prod",
		Schedule:   domain.Schedule{Type: domain.ScheduleDaily, At: "03:00", Timezone: "UTC"},
		Retention:  domain.Retention{Type: domain.RetentionAll},
	}
	job, err := svc.StartManualBackup(context.Background(), dbID, uuid.New())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	svc.runBackup(context.Background(), uuid.New(), job.ID)
	done, err := store.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != domain.BackupCompleted {
		t.Fatalf("expected completed, got %s (%s)", done.Status, done.ErrorMessage)
	}
	if done.Checksum == "" || done.StorageKey == "" || done.SizeBytes == 0 {
		t.Fatalf("artifact metadata missing: %+v", done)
	}
	if len(prov.objects) != 1 {
		t.Fatalf("expected 1 object uploaded, got %d", len(prov.objects))
	}
}

func TestLatestRetentionDeletesOlderObject(t *testing.T) {
	svc, store, prov, _, _ := newService()
	dbID := uuid.New()
	destID := uuid.New()
	store.config = &domain.BackupConfiguration{
		ID: uuid.New(), DatabaseID: dbID, DestinationID: destID, Enabled: true,
		PathPrefix: "databases/prod",
		Schedule:   domain.Schedule{Type: domain.ScheduleDaily, At: "03:00", Timezone: "UTC"},
		Retention:  domain.Retention{Type: domain.RetentionLatest},
	}
	// first backup
	j1, _ := svc.StartManualBackup(context.Background(), dbID, uuid.New())
	svc.runBackup(context.Background(), uuid.New(), j1.ID)
	// second backup
	j2, _ := svc.StartManualBackup(context.Background(), dbID, uuid.New())
	svc.runBackup(context.Background(), uuid.New(), j2.ID)

	if len(prov.objects) != 1 {
		t.Fatalf("latest retention should keep exactly 1 object, got %d", len(prov.objects))
	}
	done2, _ := store.GetJob(context.Background(), j2.ID)
	if _, ok := prov.objects[done2.StorageKey]; !ok {
		t.Fatal("latest backup object should exist")
	}
}

func TestRestorePreflight(t *testing.T) {
	svc, store, prov, _, _ := newService()
	dbID := uuid.New()
	destID := uuid.New()
	store.config = &domain.BackupConfiguration{
		ID: uuid.New(), DatabaseID: dbID, DestinationID: destID, Enabled: true,
		PathPrefix: "databases/prod",
		Schedule:   domain.Schedule{Type: domain.ScheduleDaily, At: "03:00", Timezone: "UTC"},
		Retention:  domain.Retention{Type: domain.RetentionAll},
	}
	job, err := svc.StartManualBackup(context.Background(), dbID, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	svc.runBackup(context.Background(), uuid.New(), job.ID)
	done, _ := store.GetJob(context.Background(), job.ID)
	if done.Status != domain.BackupCompleted {
		t.Fatalf("backup not completed: %s", done.Status)
	}

	res, err := svc.RestorePreflight(context.Background(), job.ID, dbID, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ready {
		for _, c := range res.Checks {
			t.Logf("check %s ok=%v msg=%s", c.Name, c.OK, c.Message)
		}
		t.Fatalf("expected preflight ready, got not ready")
	}
	if !res.Compatible {
		t.Fatal("expected compatible engine")
	}
	if len(prov.objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(prov.objects))
	}
}

func TestRestorePreflightIncompatibleEngine(t *testing.T) {
	svc, store, _, _, _ := newService()
	dbID := uuid.New()
	destID := uuid.New()
	store.config = &domain.BackupConfiguration{
		ID: uuid.New(), DatabaseID: dbID, DestinationID: destID, Enabled: true,
		PathPrefix: "databases/prod",
		Schedule:   domain.Schedule{Type: domain.ScheduleDaily, At: "03:00", Timezone: "UTC"},
		Retention:  domain.Retention{Type: domain.RetentionAll},
	}
	job, _ := svc.StartManualBackup(context.Background(), dbID, uuid.New())
	svc.runBackup(context.Background(), uuid.New(), job.ID)

	// force engine mismatch in the job record
	done, _ := store.GetJob(context.Background(), job.ID)
	done.Engine = "mysql"
	store.UpdateJob(context.Background(), done)

	res, err := svc.RestorePreflight(context.Background(), job.ID, dbID, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if res.Compatible {
		t.Fatal("expected incompatible for mismatched engine")
	}
	if res.Ready {
		t.Fatal("expected not ready for incompatible engine")
	}
}
