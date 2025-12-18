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

// FileBackupRequest 文件备份请求
type FileBackupRequest struct {
	FilePaths       []string // 支持多个文件
	CompressMethod  string   // 压缩方式 (zstd/gzip/none)
	OSSEndpoint     string
	OSSAccessKey    string
	OSSSecretKey    string
	OSSBucket       string
	OSSObjectPrefix string
}

// FileBackup 执行文件备份
func FileBackup(req FileBackupRequest) error {
	if len(req.FilePaths) == 0 {
		return fmt.Errorf("没有指定要备份的文件")
	}

	// 获取公网IP（所有文件共享同一个IP）
	publicIP, err := ipfetcher.NewPublicIPFetcher().Fetch()
	if err != nil {
		logger.Warn("获取公网IP失败，将不使用IP前缀", "error", err)
		publicIP = ""
	}

	// 获取当前日期（用于目录结构）
	now := time.Now()
	dateStr := now.Format("20060102")
	timeStr := now.Format("20060102-150405")

	// 验证所有文件是否存在
	var validFiles []string
	for _, filePath := range req.FilePaths {
		info, err := os.Stat(filePath)
		if os.IsNotExist(err) {
			logger.Warn("文件不存在，跳过", "path", filePath)
			continue
		}
		if err != nil {
			logger.Warn("无法访问文件，跳过", "path", filePath, "error", err)
			continue
		}
		if info.IsDir() {
			logger.Warn("路径是目录而不是文件，跳过", "path", filePath)
			continue
		}
		validFiles = append(validFiles, filePath)
	}

	if len(validFiles) == 0 {
		return fmt.Errorf("没有有效的文件可以备份")
	}

	logger.Info("开始备份文件", "count", len(validFiles), "files", validFiles)

	// 根据文件数量决定处理方式
	var archivePath string
	if len(validFiles) == 1 {
		// 单个文件：直接压缩
		sourceFile := validFiles[0]
		fileName := filepath.Base(sourceFile)
		fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))

		// 根据压缩方式确定文件扩展名
		var ext string
		switch req.CompressMethod {
		case "gzip":
			ext = ".gz"
		case "zstd", "":
			ext = ".zst"
		case "none":
			ext = ""
		default:
			ext = ".zst" // 默认使用 zstd
		}
		archiveName := fmt.Sprintf("%s_%s%s", timeStr, fileNameWithoutExt, ext)
		archivePath = filepath.Join(os.TempDir(), archiveName)

		// 压缩单个文件
		compressMethod := req.CompressMethod
		if compressMethod == "" {
			compressMethod = "zstd" // 默认使用 zstd
		}
		logger.Info("正在压缩文件", "method", compressMethod, "file", sourceFile)
		if err := compress.CompressFile(sourceFile, archivePath, compressMethod); err != nil {
			logger.Error("压缩文件失败", "error", err)
			return err
		}
	} else {
		// 多个文件：打包成 tar 归档
		// 生成文件名：使用第一个文件的基础名称
		firstFileName := filepath.Base(validFiles[0])
		firstFileNameWithoutExt := strings.TrimSuffix(firstFileName, filepath.Ext(firstFileName))
		archiveBaseName := fmt.Sprintf("%s_%s_files", timeStr, firstFileNameWithoutExt)

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
		archiveName := archiveBaseName + ext
		archivePath = filepath.Join(os.TempDir(), archiveName)

		// 压缩多个文件
		compressMethod := req.CompressMethod
		if compressMethod == "" {
			compressMethod = "zstd" // 默认使用 zstd
		}
		logger.Info("正在压缩多个文件", "method", compressMethod, "count", len(validFiles))
		if err := compress.CompressFiles(validFiles, archivePath, compressMethod); err != nil {
			logger.Error("压缩文件失败", "error", err)
			return err
		}
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
		return err
	}

	// 上传成功后删除临时文件
	os.Remove(archivePath)
	logger.Info("文件备份完成", "count", len(validFiles))

	return nil
}

