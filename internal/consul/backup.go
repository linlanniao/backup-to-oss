package consul

import (
	"fmt"
	"io"

	"backup-to-oss/internal/logger"

	"github.com/hashicorp/consul/api"
)

// BackupConfig Consul snapshot 备份配置
type BackupConfig struct {
	Address   string // Consul 地址，如 http://localhost:8500
	Token     string // Consul ACL Token（可选）
	Stale     bool   // 是否允许从非 leader 节点获取快照
}

// BackupResult Consul snapshot 备份结果
type BackupResult struct {
	LastIndex uint64 // 最后索引
	Snapshot  io.ReadCloser // snapshot 数据流
}

// Backup 执行 Consul snapshot 备份
// 返回 snapshot 数据流和最后索引
func Backup(cfg BackupConfig) (*BackupResult, error) {
	// 创建 Consul API 客户端配置
	config := api.DefaultConfig()
	if cfg.Address != "" {
		config.Address = cfg.Address
	}
	if cfg.Token != "" {
		config.Token = cfg.Token
	}

	// 创建 Consul 客户端
	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建 Consul 客户端失败: %v", err)
	}

	logger.Info("正在连接 Consul", "address", config.Address)

	// 获取快照
	logger.Info("正在获取 Consul snapshot")
	queryOptions := &api.QueryOptions{
		AllowStale: cfg.Stale,
	}

	snap, qm, err := client.Snapshot().Save(queryOptions)
	if err != nil {
		return nil, fmt.Errorf("获取 Consul snapshot 失败: %v", err)
	}

	logger.Info("Snapshot 获取成功", "last_index", qm.LastIndex)

	return &BackupResult{
		LastIndex: qm.LastIndex,
		Snapshot:  snap,
	}, nil
}

