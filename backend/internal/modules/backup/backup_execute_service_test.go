package logic

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBackupLogEntry(t *testing.T) {
	entry := backupLogEntry("WARN", "远程任务轮询失败")
	if !strings.Contains(entry, "[WARN] 远程任务轮询失败") {
		t.Fatalf("entry = %q", entry)
	}
	if !strings.HasSuffix(entry, "\n") {
		t.Fatalf("entry must end with newline: %q", entry)
	}
}

func TestSerializeBackupSummary(t *testing.T) {
	serialized, err := serializeBackupSummary(map[string]any{"snapshot_id": "abc123", "total_files_processed": float64(8)})
	if err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal([]byte(serialized), &summary); err != nil {
		t.Fatal(err)
	}
	if summary["snapshot_id"] != "abc123" || summary["total_files_processed"] != float64(8) {
		t.Fatalf("summary = %v", summary)
	}
	if empty, err := serializeBackupSummary(nil); err != nil || empty != "" {
		t.Fatalf("nil summary = %q, %v", empty, err)
	}
}
