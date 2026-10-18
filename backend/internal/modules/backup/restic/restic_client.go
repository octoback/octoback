package restic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/log_"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ResticClient 通过 Restic CLI 实现备份和还原。
type ResticClient struct {
	config   ClientConfig
	executor ResticExecutor
}

// Snapshots 查询仓库快照，统一由 provider 准备本地 Restic 环境并执行命令。
func (p *ResticClient) Snapshots(ctx context.Context) (any, error) {
	defer p.executor.Close()
	if err := p.prepareEnvironment(ctx); err != nil {
		return nil, err
	}
	args := []string{"-r", p.repositoryURL(), "snapshots", "--json"}
	output, err := p.executor.Execute(ctx, args, p.repositoryEnvironment())
	if err != nil {
		return nil, error_.NewTextError("查询 Restic 仓库快照失败: output=%s error=%v", strings.TrimSpace(string(output)), err)
	}
	var snapshots any
	if err := yaml.Unmarshal(output, &snapshots); err != nil {
		return nil, error_.NewWrapError("解析 Restic 快照失败", err)
	}
	return snapshots, nil
}

// Stats 查询整个 Restic 仓库的实际数据统计。
func (p *ResticClient) Stats(ctx context.Context) (any, error) {
	defer p.executor.Close()
	if err := p.prepareEnvironment(ctx); err != nil {
		return nil, err
	}
	output, err := p.executor.Execute(ctx, []string{"-r", p.repositoryURL(), "stats", "--json", "--mode", "raw-data"}, p.repositoryEnvironment())
	if err != nil {
		return nil, error_.NewTextError("查询 Restic 仓库统计失败: output=%s error=%v", strings.TrimSpace(string(output)), err)
	}
	var stats any
	if err := yaml.Unmarshal(output, &stats); err != nil {
		return nil, error_.NewWrapError("解析 Restic 仓库统计失败", err)
	}
	return stats, nil
}

// Check 校验 Restic 仓库完整性。
func (p *ResticClient) Check(ctx context.Context) error {
	defer p.executor.Close()
	if err := p.prepareEnvironment(ctx); err != nil {
		return err
	}
	output, err := p.executor.Execute(ctx, []string{"-r", p.repositoryURL(), "check"}, p.repositoryEnvironment())
	if err != nil {
		return error_.NewTextError("检查 Restic 仓库失败: output=%s error=%v", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// Prune 清理 Restic 仓库中不再需要的数据。
func (p *ResticClient) Prune(ctx context.Context) error {
	defer p.executor.Close()
	if err := p.prepareEnvironment(ctx); err != nil {
		return err
	}
	args := []string{"-r", p.repositoryURL(), "forget", "--prune"}
	policy := p.config.KeepPolicy
	values, _ := policy["value"].(map[string]any)
	retention := []struct {
		name string
		key  string
	}{{"--keep-last", "keepLast"}, {"--keep-hourly", "keepHourly"}, {"--keep-daily", "keepDaily"}, {"--keep-weekly", "keepWeekly"}, {"--keep-monthly", "keepMonthly"}, {"--keep-yearly", "keepYearly"}}
	configured := false
	for _, item := range retention {
		if value, ok := values[item.key].(float64); ok && value > 0 {
			args = append(args, item.name, fmt.Sprint(int(value)))
			configured = true
		}
	}
	if duration, ok := values["duration"].(string); ok && strings.TrimSpace(duration) != "" {
		unit, _ := values["unit"].(string)
		args = append(args, "--keep-within", duration+unit)
		configured = true
	}
	if !configured {
		return error_.NewTextError("未配置仓库保留策略，不能执行清理")
	}
	output, err := p.executor.Execute(ctx, args, p.repositoryEnvironment())
	if err != nil {
		return error_.NewTextError("清理 Restic 仓库失败: output=%s error=%v", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// ForgetSnapshot 删除指定 Restic 快照，并清理不再被引用的数据块。
func (p *ResticClient) ForgetSnapshot(ctx context.Context, snapshotId string) error {
	defer p.executor.Close()
	if strings.TrimSpace(snapshotId) == "" {
		return error_.NewTextError("删除 Restic 快照失败: 快照 Id 不能为空")
	}
	if err := p.prepareEnvironment(ctx); err != nil {
		return err
	}
	args := []string{"-r", p.repositoryURL(), "forget", snapshotId, "--prune"}
	output, err := p.executor.Execute(ctx, args, p.repositoryEnvironment())
	if err != nil {
		return error_.NewTextError("删除 Restic 快照失败: snapshotId=%s output=%s error=%v", snapshotId, strings.TrimSpace(string(output)), err)
	}
	return nil
}

// SnapshotBrowser 查询指定快照中的文件条目。
func (p *ResticClient) SnapshotBrowser(ctx context.Context, snapshotId string) (any, error) {
	defer p.executor.Close()
	if strings.TrimSpace(snapshotId) == "" {
		return nil, error_.NewTextError("快照 Id 不能为空")
	}
	if err := p.prepareEnvironment(ctx); err != nil {
		return nil, err
	}
	output, err := p.executor.Execute(ctx, []string{"-r", p.repositoryURL(), "ls", "--json", snapshotId}, p.repositoryEnvironment())
	if err != nil {
		return nil, error_.NewTextError("读取 Restic 快照文件失败: output=%s error=%v", strings.TrimSpace(string(output)), err)
	}
	entries := make([]map[string]any, 0)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		var entry map[string]any
		if yaml.Unmarshal([]byte(line), &entry) == nil {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

// prepareEnvironment 让具体执行器准备本地或远程 Restic 环境。
func (p *ResticClient) prepareEnvironment(ctx context.Context) error {
	if strings.TrimSpace(p.repositoryURL()) == "" || p.config.RepositoryPassword == "" {
		return error_.NewTextError("Restic 仓库地址和密码不能为空")
	}
	return p.executor.Prepare(ctx, &p.config)
}

type resticRemoteInfo struct {
	Version    string `json:"version"`
	Platform   string `json:"platform,omitempty"`
	Arch       string `json:"arch,omitempty"`
	Size       int    `json:"size"`
	UploadedAt string `json:"uploadedAt"`
}

// StringSlice 将插件配置值转换为字符串列表。
func StringSlice(value any) []string {
	var out []string
	switch values := value.(type) {
	case []any:
		for _, item := range values {
			out = append(out, fmt.Sprint(item))
		}
	case []string:
		out = values
	case string:
		for _, item := range strings.Split(values, "\n") {
			if strings.TrimSpace(item) != "" {
				out = append(out, strings.TrimSpace(item))
			}
		}
	}
	return out
}

// Restore 将快照还原到指定目标路径。
func (p *ResticClient) Restore(ctx context.Context, request RestoreRequest) (string, error) {
	defer p.executor.Close()
	if strings.TrimSpace(p.repositoryURL()) == "" || p.config.RepositoryPassword == "" || strings.TrimSpace(request.SnapshotId) == "" || strings.TrimSpace(request.TargetPath) == "" {
		p.log("ERROR", "Restic 恢复参数不完整")
		return "", error_.NewTextError("restic 还原参数不完整")
	}
	started := time.Now()
	p.log("INFO", fmt.Sprintf("Restic 恢复开始：快照 %s，目标目录 %s，选择 %d 个路径", request.SnapshotId, request.TargetPath, len(request.IncludePaths)))
	if err := p.prepareEnvironment(ctx); err != nil {
		p.log("ERROR", "准备 Restic 恢复环境失败："+err.Error())
		return "", err
	}
	executor := p.executor
	args := []string{"-r", p.repositoryURL(), "restore", request.SnapshotId, "--target", request.TargetPath}
	if p.sftpCommand() != "" {
		args = append([]string{"-o", "sftp.command=" + p.sftpCommand()}, args...)
	}
	for _, includePath := range request.IncludePaths {
		if strings.TrimSpace(includePath) != "" {
			args = append(args, "--include", includePath)
		}
	}
	log_.Logger.Info("执行 Restic 快照还原", zap.String("snapshotId", request.SnapshotId), zap.String("targetPath", request.TargetPath))
	p.log("INFO", "Restic 完整命令："+formatResticCommand(args))
	p.log("INFO", "Restic 执行环境准备完成，开始执行 restore 命令")
	output, err := executor.Execute(ctx, args, p.repositoryEnvironment())
	p.logRestoreOutput(output)
	if err != nil {
		p.log("ERROR", fmt.Sprintf("Restic 恢复命令失败，耗时 %.1f 秒：%v", time.Since(started).Seconds(), err))
		return strings.TrimSpace(string(output)), err
	}
	p.log("INFO", fmt.Sprintf("Restic 恢复命令执行完成，耗时 %.1f 秒", time.Since(started).Seconds()))
	return strings.TrimSpace(string(output)), nil
}

func (p *ResticClient) logRestoreOutput(output []byte) {
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		level, message, _, recognized := FormatResticEvent(line)
		if recognized {
			p.log(level, message)
			if level == "INFO" && strings.HasPrefix(message, "Restic 进度 ") && p.config.Progress != nil {
				var event map[string]any
				if json.Unmarshal([]byte(line), &event) == nil {
					if percent, ok := numberValue(event["percent_done"]); ok {
						p.config.Progress(int(percent*100), "Restic 恢复中")
					}
				}
			}
			continue
		}
		lowerLine := strings.ToLower(line)
		if strings.HasPrefix(lowerLine, "error:") || strings.HasPrefix(lowerLine, "fatal:") {
			p.log("ERROR", "Restic 输出："+line)
			continue
		}
		if strings.HasPrefix(lowerLine, "warning:") {
			p.log("WARN", "Restic 输出："+line)
			continue
		}
		p.log("INFO", "Restic 输出："+line)
	}
}

// Backup 准备远程 Restic 运行环境，按需初始化仓库并执行备份。
func (p *ResticClient) Backup(ctx context.Context, request BackupRequest) (*BackupResult, error) {
	defer p.executor.Close()
	started := time.Now()
	log_.Logger.Info("开始执行 restic 备份", zap.Int("sourcePathCount", len(request.SourcePaths)), zap.Int("excludePathCount", len(request.ExcludePaths)))
	p.log("INFO", fmt.Sprintf("Restic 备份开始：%d 个来源路径，%d 个排除路径", len(request.SourcePaths), len(request.ExcludePaths)))
	if len(request.SourcePaths) == 0 || p.config.RepositoryPassword == "" {
		p.log("ERROR", "Restic 备份参数不完整")
		return nil, error_.NewTextError("restic 备份参数不完整")
	}
	if err := p.prepareEnvironment(ctx); err != nil {
		p.log("ERROR", "准备 Restic 执行环境失败："+err.Error())
		return nil, err
	}
	if err := p.executor.PrepareRepository(ctx, &p.config); err != nil {
		p.log("ERROR", "准备 Restic 仓库失败："+err.Error())
		return nil, err
	}
	executor := p.executor
	args := []string{"backup", "--json"}
	if p.config.Compression != "" {
		args = append(args, "--compression", p.config.Compression)
	}
	if p.sftpCommand() != "" {
		args = append([]string{"-o", "sftp.command=" + p.sftpCommand()}, args...)
	}
	if strings.TrimSpace(request.RepositoryTag) != "" {
		args = append(args, "--tag", request.RepositoryTag)
	}
	for _, excludePath := range request.ExcludePaths {
		if strings.TrimSpace(excludePath) != "" {
			args = append(args, "--exclude", excludePath)
		}
	}
	args = append(args, request.SourcePaths...)
	log_.Logger.Info("执行 Restic 文件备份", zap.Int("sourcePathCount", len(request.SourcePaths)))
	p.log("INFO", "Restic 完整命令："+formatResticCommand(args))
	environment := append([]string{"RESTIC_REPOSITORY=" + p.repositoryURL()}, p.repositoryEnvironment()...)
	output, err := executor.Execute(ctx, args, environment)
	if err != nil {
		summary := backupSummary(output)
		p.log("ERROR", "执行 Restic 备份失败："+err.Error())
		return &BackupResult{Status: "failed", Output: strings.TrimSpace(string(output)), Summary: summary}, error_.NewTextError("执行 Restic 备份失败: output=%s error=%v", strings.TrimSpace(string(output)), err)
	}
	log_.Logger.Info("restic 备份完成", zap.Duration("duration", time.Since(started)))
	summary := backupSummary(output)
	p.log("INFO", fmt.Sprintf("Restic 备份命令执行完成，耗时 %.1f 秒", time.Since(started).Seconds()))
	snapshotId, _ := summary["snapshot_id"].(string)
	if strings.TrimSpace(snapshotId) == "" {
		p.log("ERROR", "Restic 备份完成但未返回快照 Id")
		return &BackupResult{Status: "failed", Output: strings.TrimSpace(string(output)), Summary: summary}, error_.NewTextError("Restic 备份完成但未返回快照 Id")
	}
	p.log("INFO", "Restic 备份成功，快照 Id："+snapshotId)
	return &BackupResult{Status: "success", Output: strings.TrimSpace(string(output)), Summary: summary, SnapshotId: snapshotId}, nil
}

func formatResticCommand(args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, "restic")
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func (p *ResticClient) log(level string, message string) {
	if p.config.Log != nil {
		p.config.Log(level, message)
	}
}

func backupSummary(output []byte) map[string]any {
	var summary map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		var event map[string]any
		if json.Unmarshal([]byte(line), &event) != nil {
			continue
		}
		if value, ok := event["summary"].(map[string]any); ok {
			summary = value
		}
		if event["message_type"] == "summary" {
			summary = event
		}
	}
	return summary
}

func (p *ResticClient) repositoryURL() string {
	if strings.TrimSpace(p.config.RepositoryURL) != "" {
		return strings.TrimSpace(p.config.RepositoryURL)
	}
	if p.config.RepositoryAccess == nil {
		return ""
	}
	return "sftp:" + p.config.RepositoryAccess.Username + "@" + p.config.RepositoryAccess.Host + ":" + p.config.RepositoryPath
}

func (p *ResticClient) repositoryEnvironment() []string {
	environment := []string{"RESTIC_PASSWORD=" + p.config.RepositoryPassword}
	return append(environment, p.config.RepositoryEnvironment...)
}

func (p *ResticClient) sftpCommand() string {
	if p.config.RepositoryAccess == nil {
		return ""
	}
	return "ssh -o BatchMode=yes -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new -p " + fmt.Sprint(p.config.RepositoryAccess.Port) + " " + shellQuote(p.config.RepositoryAccess.Username+"@"+p.config.RepositoryAccess.Host) + " -s sftp"
}

func resticVersionMatches(output, expected string) bool {
	expected = strings.TrimPrefix(strings.TrimSpace(expected), "v")
	for _, field := range strings.Fields(output) {
		field = strings.TrimPrefix(field, "v")
		if field == expected {
			return true
		}
	}
	return false
}

func resticPlatform(output string) (string, string, error) {
	parts := strings.Fields(output)
	if len(parts) < 2 {
		return "", "", error_.NewTextError("无法识别远程主机系统类型: %s", output)
	}
	osName := strings.ToLower(parts[0])
	arch := strings.ToLower(parts[1])
	if osName != "linux" && osName != "windows" && osName != "darwin" {
		return "", "", error_.NewTextError("暂不支持远程系统: %s", parts[0])
	}
	switch arch {
	case "x86_64", "amd64":
		arch = "amd64"
	case "aarch64", "arm64":
		arch = "arm64"
	case "i386", "i686", "386":
		arch = "386"
	default:
		return "", "", error_.NewTextError("暂不支持远程架构: %s", parts[1])
	}
	return osName, arch, nil
}

func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }

func LoadResticBinary() ([]byte, error) {
	file := strings.TrimSpace(os.Getenv("OCTO_RESTIC_BINARY"))
	if file == "" {
		return nil, nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, error_.NewWrapError("读取 restic 二进制失败", err)
	}
	return data, nil
}

// DownloadLocalResticBinary 按当前系统平台下载匹配的 Restic 二进制。
func DownloadLocalResticBinary(version string) ([]byte, error) {
	platform := runtime.GOOS + "\n" + runtime.GOARCH
	cacheDir := filepath.Join("data", "restic")
	cacheFile := filepath.Join(cacheDir, "restic-"+strings.TrimPrefix(version, "v")+"-"+runtime.GOOS+"-"+runtime.GOARCH)
	infoFile := filepath.Join(cacheDir, "restic.info")
	var info resticRemoteInfo
	if infoData, readErr := os.ReadFile(infoFile); readErr == nil && json.Unmarshal(infoData, &info) == nil &&
		info.Version == strings.TrimPrefix(version, "v") && info.Platform == runtime.GOOS && info.Arch == runtime.GOARCH {
		if data, readErr := os.ReadFile(cacheFile); readErr == nil && len(data) > 0 {
			log_.Logger.Info("复用本地 restic 二进制", zap.String("path", cacheFile), zap.String("version", info.Version), zap.String("platform", runtime.GOOS+"/"+runtime.GOARCH))
			return data, nil
		}
	}
	log_.Logger.Info("本地 restic 缓存不存在或不匹配，开始下载", zap.String("path", cacheFile), zap.String("version", strings.TrimPrefix(version, "v")), zap.String("platform", runtime.GOOS+"/"+runtime.GOARCH))
	data, err := downloadResticBinary(platform, version)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		_ = os.WriteFile(cacheFile, data, 0o755)
		infoData, _ := json.Marshal(resticRemoteInfo{Version: strings.TrimPrefix(version, "v"), Platform: runtime.GOOS, Arch: runtime.GOARCH, Size: len(data), UploadedAt: time.Now().Format(time.RFC3339)})
		_ = os.WriteFile(infoFile, infoData, 0o644)
	}
	return data, nil
}
