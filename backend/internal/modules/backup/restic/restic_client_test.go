package restic

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	sshaccess "handfree-work/octo-backup/internal/plugins/access/ssh"
)

type recordingExecutor struct {
	args                  []string
	env                   []string
	events                []string
	repositoryAccessError error
}

func (e *recordingExecutor) Prepare(context.Context, *ClientConfig) error {
	e.events = append(e.events, "prepare")
	return nil
}
func (e *recordingExecutor) PrepareRepositoryAccess(context.Context, *ClientConfig) error {
	e.events = append(e.events, "check-repository-access")
	return e.repositoryAccessError
}
func (e *recordingExecutor) PrepareRepository(context.Context, *ClientConfig) error {
	e.events = append(e.events, "prepare-repository")
	return nil
}
func (e *recordingExecutor) Close() error { return nil }
func (e *recordingExecutor) Execute(_ context.Context, args, env []string) ([]byte, error) {
	e.events = append(e.events, "execute")
	e.args = args
	e.env = env
	return []byte(`{"total_size":1}`), nil
}

func TestSnapshotsChecksSFTPAccessBeforeExecutingRestic(t *testing.T) {
	config := ClientConfig{
		Environment:        LocalResticEnvironment,
		RepositoryAccess:   &sshaccess.SshAccess{Host: "repo.example.com", Username: "backup", Port: 22},
		RepositoryPath:     "/backups",
		RepositoryPassword: "repo-password",
	}
	client := NewResticClient(nil, config)
	executor := &recordingExecutor{}
	client.executor = executor

	if _, err := client.Snapshots(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantEvents := []string{"prepare", "check-repository-access", "execute"}
	if !reflect.DeepEqual(executor.events, wantEvents) {
		t.Fatalf("operation order = %#v, want %#v", executor.events, wantEvents)
	}
}

func TestSnapshotsStopsWhenSFTPAccessCheckFails(t *testing.T) {
	config := ClientConfig{
		Environment:        LocalResticEnvironment,
		RepositoryAccess:   &sshaccess.SshAccess{Host: "repo.example.com", Username: "backup", Port: 22},
		RepositoryPath:     "/backups",
		RepositoryPassword: "repo-password",
	}
	client := NewResticClient(nil, config)
	executor := &recordingExecutor{repositoryAccessError: errors.New("connection refused")}
	client.executor = executor

	if _, err := client.Snapshots(context.Background()); err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("Snapshots() error = %v, want SFTP connection failure", err)
	}
	if len(executor.events) != 2 || !reflect.DeepEqual(executor.events, []string{"prepare", "check-repository-access"}) {
		t.Fatalf("Restic executed after SFTP check failed: %#v", executor.events)
	}
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

func TestSnapshotStatsUsesRestoreSizeMode(t *testing.T) {
	config := ClientConfig{Environment: LocalResticEnvironment, RepositoryURL: "s3:https://example.test/repo", RepositoryPassword: "password"}
	client := NewResticClient(nil, config)
	executor := &recordingExecutor{}
	client.executor = executor
	if _, err := client.SnapshotStats(context.Background(), "snapshot-123"); err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"-r", config.RepositoryURL, "stats", "snapshot-123", "--json", "--mode", "restore-size"}
	if !reflect.DeepEqual(executor.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", executor.args, wantArgs)
	}
}

func TestLogRestoreOutputFormatsEventsAndClassifiesText(t *testing.T) {
	var entries []string
	client := &ResticClient{config: ClientConfig{Log: func(level, message string) {
		entries = append(entries, level+":"+message)
	}}}
	client.logRestoreOutput([]byte("restoring <data/file.txt>\nwarning: skipped special file\nFatal: permission denied\n"))
	want := []string{
		"INFO:Restic 输出：restoring <data/file.txt>",
		"WARN:Restic 输出：warning: skipped special file",
		"ERROR:Restic 输出：Fatal: permission denied",
	}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("日志级别和内容 = %#v, want %#v", entries, want)
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

func TestRestoreIncludesRepositoryLocation(t *testing.T) {
	config := ClientConfig{Environment: LocalResticEnvironment, RepositoryURL: "s3:https://example.test/repo", RepositoryPassword: "password"}
	client := NewResticClient(nil, config)
	executor := &recordingExecutor{}
	client.executor = executor
	if _, err := client.Restore(context.Background(), RestoreRequest{SnapshotId: "snapshot-123", TargetPath: "/restore", IncludePaths: []string{"/etc"}}); err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"-r", config.RepositoryURL, "restore", "snapshot-123", "--target", "/restore", "--include", "/etc"}
	if !reflect.DeepEqual(executor.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", executor.args, wantArgs)
	}
}

func TestFormatResticCommandQuotesArguments(t *testing.T) {
	command := formatResticCommand([]string{"-r", "sftp:user@example:/backup path", "restore", "snapshot-1", "--target", "/restore path"})
	want := "restic '-r' 'sftp:user@example:/backup path' 'restore' 'snapshot-1' '--target' '/restore path'"
	if command != want {
		t.Fatalf("command = %q, want %q", command, want)
	}
}
