package restic

import (
	"context"
	"reflect"
	"testing"
)

type recordingExecutor struct {
	args []string
	env  []string
}

func (e *recordingExecutor) Prepare(context.Context, *ClientConfig) error           { return nil }
func (e *recordingExecutor) PrepareRepository(context.Context, *ClientConfig) error { return nil }
func (e *recordingExecutor) Close() error                                           { return nil }
func (e *recordingExecutor) Execute(_ context.Context, args, env []string) ([]byte, error) {
	e.args = args
	e.env = env
	return []byte("[]"), nil
}

func TestAliOSSRepositoryConfigUsesS3URLAndCredentials(t *testing.T) {
	config := ClientConfig{
		Environment:           LocalResticEnvironment,
		RepositoryURL:         "s3:https://oss-cn-hangzhou.aliyuncs.com/archive/backups",
		RepositoryPassword:    "repo-password",
		RepositoryEnvironment: []string{"AWS_ACCESS_KEY_ID=access-key", "AWS_SECRET_ACCESS_KEY=secret-key"},
	}
	client := NewResticClient(nil, config)
	executor := &recordingExecutor{}
	client.executor = executor
	if _, err := client.Snapshots(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"-r", config.RepositoryURL, "snapshots", "--json"}
	if !reflect.DeepEqual(executor.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", executor.args, wantArgs)
	}
	wantEnvironment := []string{"RESTIC_PASSWORD=repo-password", "AWS_ACCESS_KEY_ID=access-key", "AWS_SECRET_ACCESS_KEY=secret-key"}
	if !reflect.DeepEqual(executor.env, wantEnvironment) {
		t.Fatalf("environment = %#v, want %#v", executor.env, wantEnvironment)
	}
}

func TestBackupSummaryExtractsSnapshotId(t *testing.T) {
	result := backupSummary([]byte(`{"message_type":"summary","snapshot_id":"abc123","total_files_processed":2}`))
	if result["snapshot_id"] != "abc123" {
		t.Fatalf("snapshot_id = %v, want abc123", result["snapshot_id"])
	}
}

func TestForgetSnapshotPrunesSelectedSnapshot(t *testing.T) {
	config := ClientConfig{Environment: LocalResticEnvironment, RepositoryURL: "s3:https://example.test/repo", RepositoryPassword: "password"}
	client := NewResticClient(nil, config)
	executor := &recordingExecutor{}
	client.executor = executor
	if err := client.ForgetSnapshot(context.Background(), "snapshot-123"); err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"-r", config.RepositoryURL, "forget", "snapshot-123", "--prune"}
	if !reflect.DeepEqual(executor.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", executor.args, wantArgs)
	}
}

func TestForgetSnapshotRejectsEmptyId(t *testing.T) {
	client := NewResticClient(nil, ClientConfig{Environment: LocalResticEnvironment})
	if err := client.ForgetSnapshot(context.Background(), " "); err == nil {
		t.Fatal("ForgetSnapshot() error = nil, want validation error")
	}
}
