package controller

import (
	"errors"
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
	BackupPassword  string
	KeepBackupFiles bool // 是否保留备份文件
	OSSEndpoint     string
	OSSRegion       string
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

	var backupErr error
	for i, dirPath := range req.DirPaths {
		logger.Info("开始备份目录", "index", i+1, "total", len(req.DirPaths), "path", dirPath)
		if err := backupDir(req, dirPath, publicIP, dateStr, timeStr); err != nil {
			logger.Error("目录备份失败", "path", dirPath, "error", err)
			backupErr = errors.Join(backupErr, fmt.Errorf("%s: %w", dirPath, err))
		}
	}

	return backupErr
}

func backupDir(req DirBackupRequest, dirPath, publicIP, dateStr, timeStr string) error {
	// 生成文件名：将目录路径转换为文件名格式（斜杠替换为下划线）
	dirPathForName := strings.Trim(dirPath, "/")
	dirPathForName = strings.ReplaceAll(dirPathForName, "/", "_")
	if dirPathForName == "" {
		dirPathForName = "backup"
	}

	ext := compress.Extension(req.CompressMethod, true, req.BackupPassword != "")
	archiveName := fmt.Sprintf("%s_%s%s", timeStr, dirPathForName, ext)
	workDir, err := os.MkdirTemp("", "backup-to-oss-")
	if err != nil {
		return fmt.Errorf("创建备份临时目录失败: %w", err)
	}
	keepBackup := false
	defer func() {
		if !keepBackup {
			os.RemoveAll(workDir)
		}
	}()
	archivePath := filepath.Join(workDir, archiveName)

	// 压缩目录
	compressMethod := req.CompressMethod
	if compressMethod == "" {
		compressMethod = "zstd" // 默认使用 zstd
	}
	logger.Info("正在处理目录备份", "method", compressMethod, "encrypted", req.BackupPassword != "")
	if len(req.ExcludePatterns) > 0 {
		logger.Info("排除模式", "patterns", req.ExcludePatterns)
	}
	if err := compress.CompressDir(dirPath, archivePath, req.ExcludePatterns, compressMethod, req.BackupPassword); err != nil {
		return fmt.Errorf("处理目录备份失败: %w", err)
	}
	keepBackup = req.KeepBackupFiles

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
		Region:       req.OSSRegion,
		AccessKey:    req.OSSAccessKey,
		SecretKey:    req.OSSSecretKey,
		Bucket:       req.OSSBucket,
		ObjectPrefix: objectPrefix,
	}

	if err := oss.UploadFile(archivePath, ossConfig); err != nil {
		return fmt.Errorf("上传到 OSS 失败: %w", err)
	}

	// 上传成功后根据配置决定是否删除临时文件
	if req.KeepBackupFiles {
		logger.Info("目录备份完成，备份文件已保留", "path", dirPath, "backup_file", archivePath)
	} else {
		logger.Info("目录备份完成", "path", dirPath)
	}
	return nil
}
