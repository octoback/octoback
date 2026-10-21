package restic

import (
	"bytes"
	"context"
	"fmt"
	"go.uber.org/zap"
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/log_"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LocalResticExecutor 在本机进程中执行 Restic 命令。
type LocalResticExecutor struct {
	BinaryPath       string
	AgentEnvironment []string
}

// CheckRepositoryAccess 在本机验证 SFTP 仓库认证，并按需保存仓库私钥。
func (e *LocalResticExecutor) CheckRepositoryAccess(ctx context.Context, config *ClientConfig) error {
	startedAt := time.Now()
	access := config.RepositoryAccess
	if access == nil {
		return nil
	}
	if access.Password == "" {
		if strings.TrimSpace(access.PrivateKey) == "" {
			return error_.NewTextError("SFTP SSH 授权必须配置密码或私钥")
		}
		keyPath, err := ensureLocalRepositoryKey(config.RepositoryAccessId, access.PrivateKey)
		if err != nil {
			return err
		}
		if err := e.ensureAgent(ctx); err != nil {
			return err
		}
		command := exec.CommandContext(ctx, "ssh-add", keyPath)
		command.Env = append(os.Environ(), e.AgentEnvironment...)
		command.Stdin = bytes.NewReader([]byte(access.PrivateKeyPassword + "\n"))
		if output, err := command.CombinedOutput(); err != nil {
			return error_.NewTextError("本机将 SFTP 私钥加入 ssh-agent 失败: %s", strings.TrimSpace(string(output)))
		}
	}
	command := exec.CommandContext(ctx, "ssh", repositorySftpArguments(config)...)
	command.Stdin = bytes.NewReader(nil)
	command.Stdout = io.Discard
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			err = fmt.Errorf("%w: %s", err, detail)
		}
		return repositorySftpConnectionError(config, err)
	}
	if config.Log != nil {
		config.Log("INFO", fmt.Sprintf("本机到 SFTP 仓库连接测试成功，耗时 %.1f 秒", time.Since(startedAt).Seconds()))
	}
	return nil
}

func (e *LocalResticExecutor) ensureAgent(ctx context.Context) error {
	if strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK")) != "" {
		return nil
	}
	output, err := exec.CommandContext(ctx, "ssh-agent", "-s").Output()
	if err != nil {
		return error_.NewWrapError("本机启动 ssh-agent 失败", err)
	}
	for _, line := range strings.Split(string(output), ";") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SSH_AUTH_SOCK=") || strings.HasPrefix(line, "SSH_AGENT_PID=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				value := strings.Trim(parts[1], "\"\r\n ")
				os.Setenv(parts[0], value)
				e.AgentEnvironment = append(e.AgentEnvironment, parts[0]+"="+value)
			}
		}
	}
	if strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK")) == "" {
		return error_.NewTextError("本机启动 ssh-agent 后未获得有效连接")
	}
	return nil
}

func ensureLocalRepositoryKey(accessId int64, privateKey string) (string, error) {
	if accessId <= 0 {
		return "", error_.NewTextError("SFTP 仓库 SSH 授权 Id 无效")
	}
	keyPath := filepath.FromSlash(repositoryPrivateKeyPath(accessId))
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return "", error_.NewWrapError("创建本机 SSH 密钥目录失败", err)
	}
	currentKey, statErr := os.ReadFile(keyPath)
	if os.IsNotExist(statErr) || !bytes.Equal(currentKey, []byte(privateKey)) {
		if err := os.WriteFile(keyPath, []byte(privateKey), 0600); err != nil {
			return "", error_.NewWrapError("写入本机 SFTP 私钥文件失败", err)
		}
	} else if statErr != nil {
		return "", error_.NewWrapError("检查本机 SFTP 私钥文件失败", statErr)
	}
	if err := os.Chmod(keyPath, 0600); err != nil {
		return "", error_.NewWrapError("设置本机 SFTP 私钥权限失败", err)
	}
	return keyPath, nil
}

func repositorySftpArguments(config *ClientConfig) []string {
	access := config.RepositoryAccess
	arguments := []string{"-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new", "-p", fmt.Sprint(access.Port)}
	return append(arguments, access.Username+"@"+access.Host, "-s", "sftp")
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
	command.Env = append(os.Environ(), append(e.AgentEnvironment, environment...)...)
	return command.CombinedOutput()
}
