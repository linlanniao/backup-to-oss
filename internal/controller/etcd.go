package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backup-to-oss/internal/compress"
	"backup-to-oss/internal/etcd"
	"backup-to-oss/internal/ipfetcher"
	"backup-to-oss/internal/logger"
	"backup-to-oss/internal/oss"
)

// EtcdBackupRequest etcd snapshot 备份请求
type EtcdBackupRequest struct {
	Endpoints       []string      // etcd 服务器地址列表
	CACert          string        // CA 证书文件路径（可选）
	Cert            string        // 客户端证书文件路径（可选）
	Key             string        // 客户端私钥文件路径（可选）
	User            string        // etcd 用户名（可选）
	Password        string        // etcd 密码（可选）
	DialTimeout     time.Duration // 连接超时时间
	CommandTimeout  time.Duration // 命令超时时间（0 表示无超时）
	CompressMethod  string        // 压缩方式 (zstd/gzip/none)
	OSSEndpoint     string
	OSSAccessKey    string
	OSSSecretKey    string
	OSSBucket       string
	OSSObjectPrefix string
}

// EtcdBackup 执行 etcd snapshot 备份
func EtcdBackup(req EtcdBackupRequest) error {
	// 创建上下文（如果设置了命令超时）
	ctx := context.Background()
	var cancel context.CancelFunc
	if req.CommandTimeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), req.CommandTimeout)
		defer cancel()
	} else {
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
	}

	// 创建临时文件用于保存 snapshot
	now := time.Now()
	timeStr := now.Format("20060102-150405")
	snapshotName := fmt.Sprintf("etcd-snapshot-%s.db", timeStr)
	tempSnapshotPath := filepath.Join(os.TempDir(), snapshotName)

	logger.Info("正在获取 etcd snapshot", "path", tempSnapshotPath)

	// 调用 etcd 包执行备份
	backupCfg := etcd.BackupConfig{
		Endpoints:      req.Endpoints,
		CACert:         req.CACert,
		Cert:           req.Cert,
		Key:            req.Key,
		User:           req.User,
		Password:       req.Password,
		DialTimeout:    req.DialTimeout,
		CommandTimeout: req.CommandTimeout,
	}

	result, err := etcd.Backup(ctx, backupCfg, tempSnapshotPath)
	if err != nil {
		return err
	}
	defer os.Remove(tempSnapshotPath)

	// 压缩 snapshot 文件
	compressMethod := req.CompressMethod
	if compressMethod == "" {
		compressMethod = "zstd" // 默认使用 zstd
	}

	// 根据压缩方式确定文件扩展名
	var ext string
	switch compressMethod {
	case "gzip":
		ext = ".gz"
	case "zstd":
		ext = ".zst"
	case "none":
		ext = ""
	default:
		ext = ".zst" // 默认使用 zstd
	}
	compressedPath := tempSnapshotPath + ext
	logger.Info("正在压缩 snapshot 文件", "method", compressMethod)
	if err := compress.CompressFile(tempSnapshotPath, compressedPath, compressMethod); err != nil {
		return fmt.Errorf("压缩 snapshot 文件失败: %v", err)
	}
	defer os.Remove(compressedPath)

	// 获取压缩后的文件大小
	fileInfo, err := os.Stat(tempSnapshotPath)
	if err == nil {
		compressedInfo, err := os.Stat(compressedPath)
		if err == nil {
			originalSizeMB := float64(fileInfo.Size()) / (1024 * 1024)
			compressedSizeMB := float64(compressedInfo.Size()) / (1024 * 1024)
			ratio := float64(compressedInfo.Size()) / float64(fileInfo.Size()) * 100
			logger.Info("压缩完成",
				"original_size_mb", fmt.Sprintf("%.2f", originalSizeMB),
				"compressed_size_mb", fmt.Sprintf("%.2f", compressedSizeMB),
				"compression_ratio", fmt.Sprintf("%.1f%%", ratio))
		}
	}

	// 获取公网IP（用于 OSS 路径）
	publicIP, err := ipfetcher.NewPublicIPFetcher().Fetch()
	if err != nil {
		logger.Warn("获取公网IP失败，将不使用IP前缀", "error", err)
		publicIP = ""
	}

	// 构建OSS对象前缀：{prefix}/{ip}/{date}/
	dateStr := now.Format("20060102")
	objectPrefix := req.OSSObjectPrefix
	if publicIP != "" {
		if objectPrefix != "" {
			if !strings.HasSuffix(objectPrefix, "/") {
				objectPrefix += "/"
			}
		}
		objectPrefix += publicIP + "/" + dateStr + "/"
	} else {
		if objectPrefix != "" {
			if !strings.HasSuffix(objectPrefix, "/") {
				objectPrefix += "/"
			}
		}
		objectPrefix += dateStr + "/"
	}

	// 上传到OSS
	logger.Info("正在上传 snapshot 到 OSS")
	ossConfig := oss.Config{
		Endpoint:     req.OSSEndpoint,
		AccessKey:    req.OSSAccessKey,
		SecretKey:    req.OSSSecretKey,
		Bucket:       req.OSSBucket,
		ObjectPrefix: objectPrefix,
	}

	if err := oss.UploadFile(compressedPath, ossConfig); err != nil {
		return fmt.Errorf("上传到 OSS 失败: %v", err)
	}

	logger.Info("etcd snapshot 备份完成", "version", result.Version)
	return nil
}

