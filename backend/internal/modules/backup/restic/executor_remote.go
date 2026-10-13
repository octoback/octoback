package restic

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/log_"
	sshaccess "handfree-work/octo-backup/internal/plugins/access/ssh"
)

// RemoteResticExecutor 通过已连接的 SSH 主机执行 Restic 命令。
type RemoteResticExecutor struct {
	Access     *sshaccess.SshAccess
	BinaryPath string
	Progress   func(progress int, stage string)
	Log        func(level string, message string)
	RemoteDir  string
}

func repositoryURL(config *ClientConfig) string {
	if strings.TrimSpace(config.RepositoryURL) != "" {
		return strings.TrimSpace(config.RepositoryURL)
	}
	if config.RepositoryAccess == nil {
		return ""
	}
	return "sftp:" + config.RepositoryAccess.Username + "@" + config.RepositoryAccess.Host + ":" + config.RepositoryPath
}

// Prepare 建立远程 SSH 执行连接，并确认远程主机可以执行基础命令。
func (e *RemoteResticExecutor) Prepare(ctx context.Context, config *ClientConfig) error {
	e.log("INFO", "开始准备远程 Restic 执行环境")
	if e.Access == nil {
		return error_.NewTextError("远程 Restic 执行主机不能为空")
	}
	if err := e.Access.Connect(ctx); err != nil {
		return error_.NewWrapError("连接远程 Restic 执行主机失败", err)
	}
	e.log("INFO", "远程 Restic 执行主机连接成功")
	if _, err := e.Access.Execute(ctx, "printf restic-remote-ready"); err != nil {
		_ = e.Access.Close()
		return error_.NewWrapError("远程 Restic 执行环境不可用", err)
	}
	remoteDirectory := strings.TrimSpace(config.RemoteDirectory)
	if remoteDirectory == "" {
		remoteDirectory = ".octo_backup"
	}
	e.RemoteDir = remoteDirectory
	binaryPath := path.Join(remoteDirectory, "restic")
	infoPath := path.Join(remoteDirectory, "restic.info")
	infoOutput, infoErr := e.Access.Execute(ctx, "cat "+shellQuote(infoPath))
	var info resticRemoteInfo
	infoValid := infoErr == nil && json.Unmarshal(infoOutput, &info) == nil && resticVersionMatches(info.Version, config.Version)
	binaryOutput, binaryErr := e.Access.Execute(ctx, "if [ -x "+shellQuote(binaryPath)+" ]; then printf yes; else printf no; fi")
	if infoValid && binaryErr == nil && strings.TrimSpace(string(binaryOutput)) == "yes" {
		e.BinaryPath = binaryPath
		config.ResticBinaryPath = binaryPath
		log_.Logger.Info("复用远程 restic 二进制", zap.String("path", binaryPath), zap.String("version", info.Version))
		e.log("INFO", "复用远程 Restic 二进制："+binaryPath)
		return nil
	}
	osName, arch, err := e.Access.Platform(ctx)
	if err != nil {
		return error_.NewWrapError("探测远程 Restic 执行主机平台失败", err)
	}
	config.ResticBinary, err = downloadResticBinary(osName+"\n"+arch, config.Version)
	if err != nil {
		return error_.NewWrapError("准备远程 Restic 二进制失败", err)
	}
	if err := e.Access.Upload(ctx, binaryPath, config.ResticBinary, 0700); err != nil {
		return error_.NewWrapError("上传远程 Restic 二进制失败", err)
	}
	e.log("INFO", "远程 Restic 二进制上传完成："+binaryPath)
	infoData, err := json.Marshal(resticRemoteInfo{Version: strings.TrimPrefix(config.Version, "v"), Size: len(config.ResticBinary), UploadedAt: time.Now().Format(time.RFC3339)})
	if err != nil {
		return error_.NewWrapError("生成远程 Restic 信息文件失败", err)
	}
	if err := e.Access.Upload(ctx, infoPath, infoData, 0600); err != nil {
		return error_.NewWrapError("写入远程 Restic 信息文件失败", err)
	}
	config.ResticBinaryPath = binaryPath
	e.BinaryPath = binaryPath
	e.log("INFO", "远程 Restic 执行环境准备完成")
	return nil
}

// PrepareRepository 上传仓库环境配置，并在仓库不存在时初始化仓库。
func (e *RemoteResticExecutor) PrepareRepository(ctx context.Context, config *ClientConfig) error {
	e.log("INFO", "开始准备 Restic 仓库")
	if e.Access == nil {
		return error_.NewTextError("远程 Restic 执行主机不能为空")
	}
	configPath := path.Join(strings.TrimSpace(config.RemoteDirectory), "restic.env")
	if strings.TrimSpace(config.RemoteDirectory) == "" {
		configPath = path.Join(".octo_backup", "restic.env")
	}
	env := fmt.Sprintf("RESTIC_REPOSITORY=%s\nRESTIC_PASSWORD=%s\n", shellQuote(repositoryURL(config)), shellQuote(config.RepositoryPassword))
	for _, value := range config.RepositoryEnvironment {
		key, variable, found := strings.Cut(value, "=")
		if found {
			env += key + "=" + shellQuote(variable) + "\n"
		}
	}
	if err := e.Access.Upload(ctx, configPath, []byte(env), 0600); err != nil {
		return error_.NewWrapError("上传远程 Restic 仓库配置失败", err)
	}
	binaryPath := config.ResticBinaryPath
	if binaryPath == "" {
		binaryPath = path.Join(path.Dir(configPath), "restic")
	}
	command := "set -a; . " + shellQuote(configPath) + "; set +a; " + shellQuote(binaryPath)
	if config.RepositoryAccess != nil {
		command += " -o sftp.command=" + shellQuote(repositorySftpCommand(config))
	}
	command += " snapshots >/dev/null 2>&1 || " + shellQuote(binaryPath)
	if config.RepositoryAccess != nil {
		command += " -o sftp.command=" + shellQuote(repositorySftpCommand(config))
	}
	command += " init"
	if _, err := e.Access.Execute(ctx, command); err != nil {
		return error_.NewWrapError("初始化远程 Restic 仓库失败", err)
	}
	e.log("INFO", "Restic 仓库准备完成")
	return nil
}

func repositorySftpCommand(config *ClientConfig) string {
	if config.RepositoryAccess == nil {
		return ""
	}
	return "ssh -o BatchMode=yes -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new -p " + fmt.Sprint(config.RepositoryAccess.Port) + " " + shellQuote(config.RepositoryAccess.Username+"@"+config.RepositoryAccess.Host) + " -s sftp"
}

// Execute 在远程主机执行 Restic 命令，并通过 shell 环境赋值传入变量。
func (e RemoteResticExecutor) Execute(ctx context.Context, args []string, environment []string) ([]byte, error) {
	if containsResticAction(args, "backup") {
		return e.executeBackupDetached(ctx, args, environment)
	}
	command := make([]string, 0, len(environment)+len(args)+1)
	for _, value := range environment {
		key, variable, found := strings.Cut(value, "=")
		if found {
			command = append(command, key+"="+shellQuote(variable))
		}
	}
	command = append(command, shellQuote(e.BinaryPath))
	for _, argument := range args {
		command = append(command, shellQuote(argument))
	}
	return e.Access.Execute(ctx, strings.Join(command, " "))
}

func containsResticAction(args []string, action string) bool {
	for _, arg := range args {
		if arg == action {
			return true
		}
	}
	return false
}

func (e RemoteResticExecutor) executeBackupDetached(ctx context.Context, args []string, environment []string) ([]byte, error) {
	if e.Access == nil {
		return nil, error_.NewTextError("远程 Restic 执行主机不能为空")
	}
	remoteDirectory := e.RemoteDir
	if remoteDirectory == "" {
		remoteDirectory = ".octo_backup"
	}
	jobPath := path.Join(remoteDirectory, "jobs", "restic-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	outputPath := jobPath + ".output"
	statusPath := jobPath + ".status"
	command := []string{"mkdir -p " + shellQuote(path.Dir(jobPath)) + "; rm -f " + shellQuote(outputPath) + " " + shellQuote(statusPath) + "; ("}
	for _, value := range environment {
		key, variable, found := strings.Cut(value, "=")
		if found {
			command = append(command, key+"="+shellQuote(variable))
		}
	}
	command = append(command, shellQuote(e.BinaryPath))
	for _, argument := range args {
		command = append(command, shellQuote(argument))
	}
	command = append(command, ">"+shellQuote(outputPath)+" 2>&1; code=$?; printf '%s' \"$code\" > "+shellQuote(statusPath)+") </dev/null >/dev/null 2>&1 &")
	if err := e.Access.Close(); err != nil {
		return nil, error_.NewWrapError("关闭远程 Restic 准备连接失败", err)
	}
	if err := e.Access.Connect(ctx); err != nil {
		return nil, error_.NewWrapError("启动远程 Restic 任务时连接失败", err)
	}
	if _, err := e.Access.Execute(ctx, strings.Join(command, " ")); err != nil {
		_ = e.Access.Close()
		return nil, error_.NewWrapError("启动远程 Restic 任务失败", err)
	}
	e.log("INFO", "远程 Restic 备份任务已启动")
	_ = e.Access.Close()

	reportedOffset := 0
	lastProgressMessage := ""
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
		if err := e.Access.Connect(ctx); err != nil {
			e.log("WARN", "轮询远程 Restic 任务时连接失败："+err.Error())
			return nil, error_.NewWrapError("轮询远程 Restic 任务时连接失败", err)
		}
		status, statusErr := e.Access.Execute(ctx, "cat "+shellQuote(statusPath)+" 2>/dev/null || true")
		output, outputErr := e.Access.Execute(ctx, "cat "+shellQuote(outputPath)+" 2>/dev/null || true")
		_ = e.Access.Close()
		if statusErr != nil || outputErr != nil {
			e.log("WARN", "轮询远程 Restic 任务输出失败")
			continue
		}
		lastProgressMessage = e.reportProgress(output, &reportedOffset, lastProgressMessage)
		codeText := strings.TrimSpace(string(status))
		if codeText == "" {
			continue
		}
		code, err := strconv.Atoi(codeText)
		if err != nil {
			e.log("ERROR", "远程 Restic 任务状态无效："+codeText)
			return output, error_.NewTextError("远程 Restic 任务状态无效: %s", codeText)
		}
		if code != 0 {
			e.log("ERROR", "远程 Restic 备份任务失败")
			e.cleanupRemoteJob(ctx, outputPath, statusPath)
			return output, error_.NewTextError("远程 Restic 备份失败: %s", strings.TrimSpace(string(output)))
		}
		e.cleanupRemoteJob(ctx, outputPath, statusPath)
		if e.Progress != nil {
			e.Progress(100, "备份完成")
		}
		e.log("INFO", "远程 Restic 备份任务完成")
		return output, nil
	}
}

func (e RemoteResticExecutor) cleanupRemoteJob(ctx context.Context, outputPath, statusPath string) {
	if e.Access.Connect(ctx) != nil {
		return
	}
	_, _ = e.Access.Execute(ctx, "rm -f "+shellQuote(outputPath)+" "+shellQuote(statusPath))
	_ = e.Access.Close()
}

func (e RemoteResticExecutor) reportProgress(output []byte, reportedOffset *int, lastProgressMessage string) string {
	progress := 5
	progressMessage := ""
	for _, line := range strings.Split(string(output), "\n") {
		var message struct {
			MessageType string  `json:"message_type"`
			PercentDone float64 `json:"percent_done"`
		}
		if json.Unmarshal([]byte(line), &message) == nil && message.MessageType == "status" {
			progress = int(message.PercentDone * 100)
			_, progressMessage, _, _ = FormatResticEvent(line)
		}
	}
	newOutput := string(output)
	if reportedOffset != nil {
		if *reportedOffset > len(newOutput) {
			*reportedOffset = 0
		}
		newOutput = newOutput[*reportedOffset:]
		lastNewline := strings.LastIndex(newOutput, "\n")
		if lastNewline < 0 {
			return lastProgressMessage
		}
		newOutput = newOutput[:lastNewline+1]
		*reportedOffset += len(newOutput)
	}
	statusReported := false
	for _, line := range strings.Split(newOutput, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		level, message, summary, _ := FormatResticEvent(line)
		if level == "INFO" && strings.HasPrefix(message, "Restic 进度 ") {
			progressMessage = message
			statusReported = true
			if message != lastProgressMessage {
				e.log(level, message)
			}
		} else {
			e.log(level, message)
		}
		if summary != nil {
			data, err := json.Marshal(summary)
			if err == nil {
				e.log("INFO", "Restic summary JSON："+string(data))
			}
		}
	}
	if !statusReported && progressMessage == "" {
		progressMessage = fmt.Sprintf("轮询远程 Restic 任务进度：%d%%", progress)
		if progressMessage != lastProgressMessage {
			e.log("INFO", progressMessage)
		}
	}
	if e.Progress == nil {
		return progressMessage
	}
	e.Progress(progress, "远程 Restic 备份中")
	return progressMessage
}

func (e RemoteResticExecutor) log(level string, message string) {
	if e.Log != nil && strings.TrimSpace(message) != "" {
		e.Log(level, message)
	}
}

// Close 关闭远程 SSH 执行连接。
func (e *RemoteResticExecutor) Close() error {
	if e.Access == nil {
		return nil
	}
	return e.Access.Close()
}
