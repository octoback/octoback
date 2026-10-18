package restic

import (
	"context"
	"go.uber.org/zap"
	"handfree-work/octo-backup/internal/base/log_"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// LocalResticExecutor 在本机进程中执行 Restic 命令。
type LocalResticExecutor struct {
	BinaryPath string
}

// Prepare 准备本机 Restic 二进制，不处理 SSH 连接和仓库配置。
func (e *LocalResticExecutor) Prepare(_ context.Context, config *ClientConfig) error {
	if len(config.ResticBinary) == 0 {
		binary, err := DownloadLocalResticBinary(config.Version)
		if err != nil {
			return err
		}
		config.ResticBinary = binary
	}
	path, err := localResticBinaryPath(config.Version)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, config.ResticBinary, 0o755); err != nil {
		return err
	}
	e.BinaryPath = path
	return nil
}

// PrepareRepository 本地执行器无需上传配置或初始化远程仓库。
func (e *LocalResticExecutor) PrepareRepository(_ context.Context, _ *ClientConfig) error { return nil }

// Close 释放本地执行器资源。
func (e *LocalResticExecutor) Close() error { return nil }

func localResticBinaryPath(version string) (string, error) {
	directory := filepath.Join("data", "restic")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(directory, "restic-"+strings.TrimPrefix(version, "v")+"-"+runtime.GOOS+"-"+runtime.GOARCH), nil
}

// Execute 执行本机 Restic 命令。
func (e LocalResticExecutor) Execute(ctx context.Context, args []string, environment []string) ([]byte, error) {
	log_.Logger.Info("执行本地 Restic 完整命令", zap.String("command", formatResticCommand(args)))
	command := exec.CommandContext(ctx, e.BinaryPath, args...)
	command.Env = append(os.Environ(), environment...)
	return command.CombinedOutput()
}
