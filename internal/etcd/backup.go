package etcd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"backup-to-oss/internal/logger"

	"go.etcd.io/etcd/client/pkg/v3/logutil"
	clientv3 "go.etcd.io/etcd/client/v3"
	snapshot "go.etcd.io/etcd/client/v3/snapshot"
	"go.uber.org/zap"
)

// BackupConfig etcd snapshot 备份配置
type BackupConfig struct {
	Endpoints      []string      // etcd 服务器地址列表
	CACert         string        // CA 证书文件路径（可选）
	Cert           string        // 客户端证书文件路径（可选）
	Key            string        // 客户端私钥文件路径（可选）
	User           string        // etcd 用户名（可选）
	Password       string        // etcd 密码（可选）
	DialTimeout    time.Duration // 连接超时时间
	CommandTimeout time.Duration // 命令超时时间（0 表示无超时）
}

// BackupResult etcd snapshot 备份结果
type BackupResult struct {
	Version string // etcd 版本
	Path    string // snapshot 文件路径
}

// Backup 执行 etcd snapshot 备份
// 返回备份文件的路径和版本信息
func Backup(ctx context.Context, cfg BackupConfig, snapshotPath string) (*BackupResult, error) {
	// 创建 logger
	lg, err := logutil.CreateDefaultZapLogger(zap.InfoLevel)
	if err != nil {
		return nil, fmt.Errorf("创建 logger 失败: %v", err)
	}

	// 构建 etcd 客户端配置
	clientCfg := clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
	}

	// 配置 TLS（如果提供了证书）
	if cfg.CACert != "" || cfg.Cert != "" || cfg.Key != "" {
		tlsConfig := &tls.Config{}

		// 加载 CA 证书
		if cfg.CACert != "" {
			caCert, err := os.ReadFile(cfg.CACert)
			if err != nil {
				return nil, fmt.Errorf("读取 CA 证书失败: %v", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("解析 CA 证书失败")
			}
			tlsConfig.RootCAs = caCertPool
		}

		// 加载客户端证书和私钥
		if cfg.Cert != "" && cfg.Key != "" {
			cert, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
			if err != nil {
				return nil, fmt.Errorf("加载客户端证书失败: %v", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		clientCfg.TLS = tlsConfig
	}

	// 配置用户名和密码（如果提供了）
	if cfg.User != "" && cfg.Password != "" {
		clientCfg.Username = cfg.User
		clientCfg.Password = cfg.Password
	}

	logger.Info("正在连接 etcd", "endpoints", cfg.Endpoints)

	// 使用 etcd 官方库保存 snapshot
	version, err := snapshot.SaveWithVersion(ctx, lg, clientCfg, snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("保存 etcd snapshot 失败: %v", err)
	}

	logger.Info("Snapshot 保存成功", "path", snapshotPath, "version", version)

	// 验证 snapshot 文件
	fileInfo, err := os.Stat(snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("获取 snapshot 文件信息失败: %v", err)
	}
	if fileInfo.Size() == 0 {
		return nil, fmt.Errorf("snapshot 文件为空")
	}

	logger.Info("Snapshot 验证成功", "size_bytes", fileInfo.Size(), "size_mb", fmt.Sprintf("%.2f", float64(fileInfo.Size())/(1024*1024)))

	// 执行 snapshot status 操作验证 snapshot 完整性
	logger.Info("正在执行 snapshot status 操作")
	statusInfo, err := CheckSnapshotStatus(snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("snapshot status 失败，文件可能已损坏: %v", err)
	}
	logger.Info("Snapshot status 成功",
		"hash", statusInfo.Hash,
		"revision", statusInfo.Revision,
		"total_key", statusInfo.TotalKey,
		"total_size", statusInfo.TotalSize)

	return &BackupResult{
		Version: version,
		Path:    snapshotPath,
	}, nil
}
