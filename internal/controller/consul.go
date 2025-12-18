package controller

import (
	"fmt"
	"io"
	"os"

	"path/filepath"
	"strings"
	"time"

	"backup-to-oss/internal/compress"
	"backup-to-oss/internal/ipfetcher"
	"backup-to-oss/internal/logger"
	"backup-to-oss/internal/oss"

	"github.com/hashicorp/consul/api"
	"github.com/rboyer/safeio"
)

// ConsulBackupRequest Consul snapshot 备份请求
type ConsulBackupRequest struct {
	ConsulAddress   string // Consul 地址，如 http://localhost:8500
	ConsulToken     string // Consul ACL Token（可选）
	Stale           bool   // 是否允许从非 leader 节点获取快照
	CompressMethod  string // 压缩方式 (zstd/gzip/none)
	OSSEndpoint     string
	OSSAccessKey    string
	OSSSecretKey    string
	OSSBucket       string
	OSSObjectPrefix string
}

// ConsulBackup 执行 Consul snapshot 备份
func ConsulBackup(req ConsulBackupRequest) error {
	// 创建 Consul API 客户端配置
	config := api.DefaultConfig()
	if req.ConsulAddress != "" {
		config.Address = req.ConsulAddress
	}
	if req.ConsulToken != "" {
		config.Token = req.ConsulToken
	}

	// 创建 Consul 客户端
	client, err := api.NewClient(config)
	if err != nil {
		return fmt.Errorf("创建 Consul 客户端失败: %v", err)
	}

	logger.Info("正在连接 Consul", "address", config.Address)

	// 获取快照
	logger.Info("正在获取 Consul snapshot")
	queryOptions := &api.QueryOptions{
		AllowStale: req.Stale,
	}

	snap, qm, err := client.Snapshot().Save(queryOptions)
	if err != nil {
		return fmt.Errorf("获取 Consul snapshot 失败: %v", err)
	}
	defer snap.Close()

	logger.Info("Snapshot 获取成功", "last_index", qm.LastIndex)

	// 创建临时文件用于保存 snapshot
	now := time.Now()
	timeStr := now.Format("20060102-150405")
	snapshotName := fmt.Sprintf("consul-snapshot-%s.snap", timeStr)
	tempSnapshotPath := filepath.Join(os.TempDir(), snapshotName)
	unverifiedPath := tempSnapshotPath + ".unverified"

	// 先写入未验证的文件
	logger.Info("正在保存 snapshot 到临时文件", "path", unverifiedPath)
	if _, err := safeio.WriteToFile(snap, unverifiedPath, 0600); err != nil {
		return fmt.Errorf("写入 snapshot 文件失败: %v", err)
	}
	defer os.Remove(unverifiedPath)

	// 验证 snapshot（通过读取文件来验证）
	logger.Info("正在验证 snapshot")
	f, err := os.Open(unverifiedPath)
	if err != nil {
		return fmt.Errorf("打开 snapshot 文件失败: %v", err)
	}
	defer f.Close()

	// 读取文件内容以验证（简单的文件大小检查）
	fileInfo, err := f.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %v", err)
	}
	if fileInfo.Size() == 0 {
		return fmt.Errorf("snapshot 文件为空")
	}

	// 重新定位到文件开头
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("重置文件指针失败: %v", err)
	}

	// 读取一些数据来验证文件可读性
	buf := make([]byte, 1024)
	if _, err := f.Read(buf); err != nil && err != io.EOF {
		return fmt.Errorf("读取 snapshot 文件失败: %v", err)
	}
	f.Close()

	logger.Info("Snapshot 验证成功", "size_bytes", fileInfo.Size(), "size_mb", fmt.Sprintf("%.2f", float64(fileInfo.Size())/(1024*1024)))

	// 重命名为最终文件
	if err := safeio.Rename(unverifiedPath, tempSnapshotPath); err != nil {
		return fmt.Errorf("重命名 snapshot 文件失败: %v", err)
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

	logger.Info("Consul snapshot 备份完成", "last_index", qm.LastIndex)
	return nil
}
