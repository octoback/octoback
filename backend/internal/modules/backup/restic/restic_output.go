package restic

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// FormatResticEvent 将 Restic 的 JSON 事件转换为界面可读的日志消息。
func FormatResticEvent(line string) (string, string, map[string]any, bool) {
	raw := strings.TrimSpace(line)
	if raw == "" {
		return "INFO", "", nil, false
	}
	var event map[string]any
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		return "WARN", raw, nil, false
	}
	switch fmt.Sprint(event["message_type"]) {
	case "status":
		return "INFO", formatResticStatus(event), nil, true
	case "summary":
		return "INFO", formatResticSummary(event), event, true
	case "error":
		return "ERROR", formatResticError(event), nil, true
	default:
		data, err := json.MarshalIndent(event, "", "  ")
		if err != nil {
			return "WARN", raw, nil, false
		}
		return "WARN", "Restic 未识别事件:\n" + string(data), nil, false
	}
}

func formatResticStatus(event map[string]any) string {
	parts := []string{"Restic 进度 " + formatPercent(event["percent_done"])}
	filesDone, hasFilesDone := numberValue(event["files_done"])
	totalFiles, hasTotalFiles := numberValue(event["total_files"])
	if hasFilesDone && hasTotalFiles {
		parts = append(parts, fmt.Sprintf("%.0f/%.0f 个文件", filesDone, totalFiles))
	}
	bytesDone, hasBytesDone := numberValue(event["bytes_done"])
	totalBytes, hasTotalBytes := numberValue(event["total_bytes"])
	if hasBytesDone && hasTotalBytes {
		parts = append(parts, fmt.Sprintf("%s/%s", formatBytes(bytesDone), formatBytes(totalBytes)))
	}
	if currentFile, ok := currentFileValue(event["current_files"]); ok {
		parts = append(parts, "当前文件 "+currentFile)
	}
	if remaining, ok := numberValue(event["seconds_remaining"]); ok && remaining >= 0 {
		parts = append(parts, fmt.Sprintf("预计剩余 %s", formatDuration(remaining)))
	}
	return strings.Join(parts, "，")
}

func formatResticSummary(event map[string]any) string {
	parts := []string{"Restic 备份汇总"}
	if snapshotId, ok := event["snapshot_id"].(string); ok && strings.TrimSpace(snapshotId) != "" {
		parts = append(parts, "快照 "+snapshotId)
	}
	if value, ok := numberValue(event["total_files_processed"]); ok {
		parts = append(parts, fmt.Sprintf("处理文件 %.0f 个", value))
	}
	if value, ok := numberValue(event["total_bytes_processed"]); ok {
		parts = append(parts, "处理数据 "+formatBytes(value))
	}
	if value, ok := numberValue(event["total_duration"]); ok {
		parts = append(parts, fmt.Sprintf("耗时 %.1f 秒", value))
	}
	for _, name := range []string{"files_new", "files_changed", "files_unmodified", "dirs_new", "dirs_changed", "dirs_unmodified"} {
		if value, ok := numberValue(event[name]); ok {
			parts = append(parts, fmt.Sprintf("%s %.0f", summaryLabel(name), value))
		}
	}
	return strings.Join(parts, "，")
}

func formatResticError(event map[string]any) string {
	if message, ok := event["error"].(string); ok && strings.TrimSpace(message) != "" {
		return "Restic 错误：" + message
	}
	data, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return "Restic 错误"
	}
	return "Restic 错误：" + string(data)
}

func summaryLabel(name string) string {
	labels := map[string]string{
		"files_new":        "新增文件",
		"files_changed":    "变更文件",
		"files_unmodified": "未变更文件",
		"dirs_new":         "新增目录",
		"dirs_changed":     "变更目录",
		"dirs_unmodified":  "未变更目录",
	}
	if label, ok := labels[name]; ok {
		return label
	}
	return name
}

func numberValue(value any) (float64, bool) {
	number, ok := value.(float64)
	return number, ok && !math.IsNaN(number)
}

func formatPercent(value any) string {
	number, ok := numberValue(value)
	if !ok {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", number*100)
}

func formatBytes(value float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unitIndex := 0
	for value >= 1024 && unitIndex < len(units)-1 {
		value /= 1024
		unitIndex++
	}
	return fmt.Sprintf("%.1f %s", value, units[unitIndex])
}

func formatDuration(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.0f 秒", seconds)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%.0f 分钟", minutes)
	}
	return fmt.Sprintf("%.1f 小时", minutes/60)
}

func currentFileValue(value any) (string, bool) {
	files, ok := value.([]any)
	if !ok || len(files) == 0 {
		return "", false
	}
	file, ok := files[len(files)-1].(string)
	return file, ok && strings.TrimSpace(file) != ""
}
