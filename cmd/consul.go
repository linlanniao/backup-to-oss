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
	consulAddress string
	consulToken   string
	consulStale   bool
)

// consulCmd represents the consul command
var consulCmd = &cobra.Command{
	Use:   "consul",
	Short: "备份 Consul snapshot 到 OSS",
	Long: `从 Consul 服务器获取 snapshot 并上传到阿里云 OSS。

配置可以通过以下方式提供：
1. .env 文件（可通过 --env-file 指定路径）
2. 环境变量
3. 命令行参数（优先级最高）

OSS 相关配置（--endpoint, --access-key, --secret-key, --bucket, --prefix）为全局参数，可在任何子命令中使用。

示例:
  backup-to-oss consul --address http://localhost:8500
  或
  backup-to-oss consul --address http://localhost:8500 --token your-token
  或
  backup-to-oss consul --address http://localhost:8500 --stale
  或
  backup-to-oss --env-file /path/to/.env consul --address http://localhost:8500`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runConsulBackup(); err != nil {
			logger.Error("备份失败", "error", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(consulCmd)

	consulCmd.Flags().StringVar(&consulAddress, "address", "", "Consul 服务器地址（可通过 CONSUL_ADDRESS 环境变量设置，默认为 http://127.0.0.1:8500）")
	consulCmd.Flags().StringVar(&consulToken, "token", "", "Consul ACL Token（可通过 CONSUL_TOKEN 环境变量设置，可选）")
	consulCmd.Flags().BoolVar(&consulStale, "stale", false, "允许从非 leader 节点获取快照（可通过 CONSUL_STALE 环境变量设置，设置为 true 时允许）")
}

func runConsulBackup() error {
	// 加载配置（从 .env 文件或环境变量）
	cfg, err := config.LoadConfig(envFile)
	if err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	// 从环境变量获取 Consul 配置（如果命令行参数未设置）
	consulAddr := consulAddress
	if consulAddr == "" {
		if envAddr := os.Getenv("CONSUL_ADDRESS"); envAddr != "" {
			consulAddr = envAddr
		} else {
			consulAddr = "http://127.0.0.1:8500" // 默认地址
		}
	}

	consulTok := consulToken
	if consulTok == "" {
		consulTok = os.Getenv("CONSUL_TOKEN")
	}

	consulStaleFlag := consulStale
	if !consulStaleFlag {
		if envStale := os.Getenv("CONSUL_STALE"); envStale == "true" || envStale == "1" {
			consulStaleFlag = true
		}
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
	cfg.MergeWithFlags("", "", compressMethodValue, ossEndpoint, ossAccessKey, ossSecretKey, ossBucket, ossObjectPrefix)

	// 验证 OSS 配置
	if cfg.OSSEndpoint == "" {
		return fmt.Errorf("OSS端点未设置（通过 --endpoint 参数或 OSS_ENDPOINT 环境变量）")
	}
	if cfg.OSSAccessKey == "" {
		return fmt.Errorf("OSS AccessKey未设置（通过 --access-key 参数或 OSS_ACCESS_KEY 环境变量）")
	}
	if cfg.OSSSecretKey == "" {
		return fmt.Errorf("OSS SecretKey未设置（通过 --secret-key 参数或 OSS_SECRET_KEY 环境变量）")
	}
	if cfg.OSSBucket == "" {
		return fmt.Errorf("OSS存储桶未设置（通过 --bucket 参数或 OSS_BUCKET 环境变量）")
	}

	// 构建请求
	req := controller.ConsulBackupRequest{
		ConsulAddress:   consulAddr,
		ConsulToken:     consulTok,
		Stale:           consulStaleFlag,
		CompressMethod:  cfg.CompressMethod,
		OSSEndpoint:     cfg.OSSEndpoint,
		OSSAccessKey:    cfg.OSSAccessKey,
		OSSSecretKey:    cfg.OSSSecretKey,
		OSSBucket:       cfg.OSSBucket,
		OSSObjectPrefix: cfg.OSSObjectPrefix,
	}

	return controller.ConsulBackup(req)
}
