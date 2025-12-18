package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backup-to-oss/internal/compress"
	"backup-to-oss/internal/ipfetcher"
	"backup-to-oss/internal/logger"
	"backup-to-oss/internal/oss"
)

// DirBackupRequest 目录备份请求
type DirBackupRequest struct {
	DirPaths        []string // 支持多个目录
	ExcludePatterns []string // 排除模式列表
	CompressMethod  string   // 压缩方式 (zstd/gzip/none)
	OSSEndpoint     string
	OSSAccessKey    string
	OSSSecretKey    string
	OSSBucket       string
	OSSObjectPrefix string
}

// DirBackup 执行目录备份
func DirBackup(req DirBackupRequest) error {
	if len(req.DirPaths) == 0 {
		return fmt.Errorf("没有指定要备份的目录")
	}

	// 获取公网IP（所有目录共享同一个IP）
	publicIP, err := ipfetcher.NewPublicIPFetcher().Fetch()
	if err != nil {
		logger.Warn("获取公网IP失败，将不使用IP前缀", "error", err)
		publicIP = ""
	}

	// 获取当前日期（用于目录结构）
	now := time.Now()
	dateStr := now.Format("20060102")
	timeStr := now.Format("20060102-150405")

	// 遍历每个目录进行备份
	for i, dirPath := range req.DirPaths {
		// 验证目录路径
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			logger.Warn("目录不存在，跳过", "path", dirPath)
			continue
		}

		logger.Info("开始备份目录", "index", i+1, "total", len(req.DirPaths), "path", dirPath)

		// 生成文件名：将目录路径转换为文件名格式（斜杠替换为下划线）
		dirPathForName := strings.Trim(dirPath, "/")
		dirPathForName = strings.ReplaceAll(dirPathForName, "/", "_")
		if dirPathForName == "" {
			dirPathForName = "backup"
		}

		// 根据压缩方式确定文件扩展名
		var ext string
		switch req.CompressMethod {
		case "gzip":
			ext = ".tgz"
		case "zstd", "":
			ext = ".tar.zst"
		case "none":
			ext = ".tar"
		default:
			ext = ".tar.zst" // 默认使用 zstd
		}
		archiveName := fmt.Sprintf("%s_%s%s", timeStr, dirPathForName, ext)
		archivePath := filepath.Join(os.TempDir(), archiveName)

		// 压缩目录
		compressMethod := req.CompressMethod
		if compressMethod == "" {
			compressMethod = "zstd" // 默认使用 zstd
		}
		logger.Info("正在压缩目录", "method", compressMethod)
		if len(req.ExcludePatterns) > 0 {
			logger.Info("排除模式", "patterns", req.ExcludePatterns)
		}
		if err := compress.CompressDir(dirPath, archivePath, req.ExcludePatterns, compressMethod); err != nil {
			logger.Error("压缩目录失败", "error", err)
			continue
		}

		// 获取文件大小
		fileInfo, err := os.Stat(archivePath)
		if err == nil {
			sizeMB := float64(fileInfo.Size()) / (1024 * 1024)
			logger.Info("压缩完成", "path", archivePath, "size_bytes", fileInfo.Size(), "size_mb", fmt.Sprintf("%.2f", sizeMB))
		}

		// 构建OSS对象前缀：{prefix}/{ip}/{date}/
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
		logger.Info("正在上传到OSS")
		ossConfig := oss.Config{
			Endpoint:     req.OSSEndpoint,
			AccessKey:    req.OSSAccessKey,
			SecretKey:    req.OSSSecretKey,
			Bucket:       req.OSSBucket,
			ObjectPrefix: objectPrefix,
		}

		if err := oss.UploadFile(archivePath, ossConfig); err != nil {
			logger.Error("上传到OSS失败", "error", err)
			os.Remove(archivePath) // 清理临时文件
			continue
		}

		// 上传成功后删除临时文件
		os.Remove(archivePath)
		logger.Info("目录备份完成", "path", dirPath)
	}

	logger.Info("所有备份任务完成", "total", len(req.DirPaths))
	return nil
}
