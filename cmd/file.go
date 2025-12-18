/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"backup-to-oss/internal/config"
	"backup-to-oss/internal/controller"
	"backup-to-oss/internal/logger"

	"github.com/spf13/cobra"
)

var (
	filePaths string
)

// fileCmd represents the file command
var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "压缩备份单个或多个文件到OSS",
	Long: `将指定文件压缩后上传到阿里云OSS（默认使用 zstd 压缩，可通过 --compress 参数选择压缩方式）。
支持备份单个文件或多个文件（多个文件用逗号分隔）。

配置可以通过以下方式提供：
1. .env 文件（可通过 --env-file 指定路径）
2. 环境变量
3. 命令行参数（优先级最高）

OSS 相关配置（--endpoint, --access-key, --secret-key, --bucket, --prefix）和压缩方式（--compress）为全局参数，可在任何子命令中使用。

示例:
  backup-to-oss file --path /path/to/file.txt
  或
  backup-to-oss file --path /path/to/file1.txt,/path/to/file2.txt
  或
  backup-to-oss file --path /path/to/file.txt --endpoint oss-cn-hangzhou.aliyuncs.com --bucket my-bucket
  或
  backup-to-oss --env-file /path/to/.env file --path /path/to/file.txt`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runFileBackup(); err != nil {
			logger.Error("备份失败", "error", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(fileCmd)

	fileCmd.Flags().StringVarP(&filePaths, "path", "p", "", "要备份的文件路径，支持多个文件用逗号分隔（可通过 FILES_TO_BACKUP 环境变量设置）")
}

func runFileBackup() error {
	// 加载配置（从 .env 文件或环境变量）
	cfg, err := config.LoadConfig(envFile)
	if err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	// 获取压缩方式（优先使用命令行参数，其次环境变量，最后使用默认值）
	compressMethodValue := compressMethod
	if compressMethodValue == "" {
		if envCompress := os.Getenv("COMPRESS_METHOD"); envCompress != "" {
			compressMethodValue = envCompress
		} else {
			compressMethodValue = "zstd" // 默认使用 zstd
		}
	}

	// 合并命令行参数（命令行参数优先级更高）
	cfg.MergeWithFileFlags(filePaths, compressMethodValue, ossEndpoint, ossAccessKey, ossSecretKey, ossBucket, ossObjectPrefix)

	// 验证配置
	if err := cfg.ValidateFileConfig(); err != nil {
		return err
	}

	// 构建请求
	req := controller.FileBackupRequest{
		FilePaths:       cfg.FilePaths,
		CompressMethod:  cfg.CompressMethod,
		OSSEndpoint:     cfg.OSSEndpoint,
		OSSAccessKey:    cfg.OSSAccessKey,
		OSSSecretKey:    cfg.OSSSecretKey,
		OSSBucket:       cfg.OSSBucket,
		OSSObjectPrefix: cfg.OSSObjectPrefix,
	}

	return controller.FileBackup(req)
}

