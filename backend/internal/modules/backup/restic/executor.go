package restic

import (
	"context"
)

// ResticExecutor 执行一个 Restic 命令并返回合并后的输出。
type ResticExecutor interface {
	Prepare(ctx context.Context, config *ClientConfig) error
	CheckRepositoryAccess(ctx context.Context, config *ClientConfig) error
	PrepareRepository(ctx context.Context, config *ClientConfig) error
	Execute(ctx context.Context, args []string, environment []string) ([]byte, error)
	Close() error
}
