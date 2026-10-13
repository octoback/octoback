package restic

import (
	"strings"
	"testing"
)

func TestFormatResticEventStatus(t *testing.T) {
	level, message, summary, ok := FormatResticEvent(`{"message_type":"status","percent_done":0.25,"files_done":2,"total_files":8,"bytes_done":1024,"total_bytes":4096,"current_files":["/var/lib/data.db"]}`)
	if !ok {
		t.Fatal("status event should be recognized")
	}
	if level != "INFO" {
		t.Fatalf("level = %q, want INFO", level)
	}
	if summary != nil {
		t.Fatalf("status summary = %v, want nil", summary)
	}
	for _, expected := range []string{"25.0%", "2/8 个文件", "1.0 KB/4.0 KB", "/var/lib/data.db"} {
		if !containsText(message, expected) {
			t.Fatalf("message = %q, want %q", message, expected)
		}
	}
}

func TestFormatResticEventSummary(t *testing.T) {
	level, message, summary, ok := FormatResticEvent(`{"message_type":"summary","snapshot_id":"abc123","files_new":2,"total_files_processed":8,"total_bytes_processed":4096,"total_duration":12.5}`)
	if !ok {
		t.Fatal("summary event should be recognized")
	}
	if level != "INFO" || summary["snapshot_id"] != "abc123" {
		t.Fatalf("level or summary is incorrect: %q, %v", level, summary)
	}
	for _, expected := range []string{"备份汇总", "abc123", "处理文件 8 个", "处理数据 4.0 KB", "耗时 12.5 秒"} {
		if !containsText(message, expected) {
			t.Fatalf("message = %q, want %q", message, expected)
		}
	}
}

func TestFormatResticEventErrorAndInvalid(t *testing.T) {
	level, message, summary, ok := FormatResticEvent(`{"message_type":"error","error":"permission denied"}`)
	if !ok || level != "ERROR" || summary != nil || !containsText(message, "permission denied") {
		t.Fatalf("error event = %q, %q, %v, %v", level, message, summary, ok)
	}
	level, message, summary, ok = FormatResticEvent(`not-json`)
	if ok || level != "WARN" || summary != nil || !containsText(message, "not-json") {
		t.Fatalf("invalid event = %q, %q, %v, %v", level, message, summary, ok)
	}
}

func TestReportProgressConsumesCompleteLinesAndDeduplicatesStatus(t *testing.T) {
	var logged []string
	executor := RemoteResticExecutor{Log: func(level, message string) {
		logged = append(logged, level+":"+message)
	}}
	statusLine := `{"message_type":"status","percent_done":0.25,"files_done":1,"total_files":4}`
	output := []byte(statusLine[:len(statusLine)-3])
	offset := 0
	lastProgress := executor.reportProgress(output, &offset, "")
	if len(logged) != 0 || offset != 0 {
		t.Fatalf("partial line was consumed: logged=%v offset=%d", logged, offset)
	}

	output = append(output, []byte(statusLine[len(statusLine)-3:]+"\n")...)
	lastProgress = executor.reportProgress(output, &offset, lastProgress)
	if len(logged) != 1 || !strings.Contains(logged[0], "25.0%") {
		t.Fatalf("first status logs = %v", logged)
	}

	output = append(output, []byte(statusLine+"\n")...)
	executor.reportProgress(output, &offset, lastProgress)
	if len(logged) != 1 {
		t.Fatalf("duplicate status was logged: %v", logged)
	}
}

func containsText(value, expected string) bool {
	return len(value) >= len(expected) && stringIndex(value, expected) >= 0
}

func stringIndex(value, expected string) int {
	for index := 0; index+len(expected) <= len(value); index++ {
		if value[index:index+len(expected)] == expected {
			return index
		}
	}
	return -1
}
