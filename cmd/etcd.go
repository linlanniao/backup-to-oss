/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"backup-to-oss/internal/config"
	"backup-to-oss/internal/controller"
	"backup-to-oss/internal/logger"

	"github.com/spf13/cobra"
)

var (
	etcdEndpoints      string
	etcdCACert         string
	etcdCert           string
	etcdKey            string
	etcdUser           string
	etcdPassword       string
	etcdDialTimeout    string
	etcdCommandTimeout string
)

// etcdCmd represents the etcd command
var etcdCmd = &cobra.Command{
	Use:   "etcd",
	Short: "备份 etcd snapshot 到 OSS",
	Long: `从 etcd 服务器获取 snapshot 并上传到阿里云 OSS。
对标 etcdctl snapshot save 命令的功能。

配置可以通过以下方式提供：
1. .env 文件（可通过 --env-file 指定路径）
2. 环境变量
3. 命令行参数（优先级最高）

OSS 相关配置（--endpoint, --access-key, --secret-key, --bucket, --prefix）为全局参数，可在任何子命令中使用。

示例:
  backup-to-oss etcd --etcd-endpoints http://127.0.0.1:2379
  或
  backup-to-oss etcd --etcd-endpoints https://127.0.0.1:2379 --cacert /etc/etcd/ca.crt --cert /etc/etcd/etcd.crt --key /etc/etcd/etcd.key
  或
  backup-to-oss etcd --etcd-endpoints http://127.0.0.1:2379 --user root --password password123
  或
  backup-to-oss etcd --etcd-endpoints https://127.0.0.1:2379 --dial-timeout 20s --command-timeout 60s
  或
  backup-to-oss --env-file /path/to/.env etcd --etcd-endpoints http://127.0.0.1:2379`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runEtcdBackup(); err != nil {
			logger.Error("备份失败", "error", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(etcdCmd)

	etcdCmd.Flags().StringVar(&etcdEndpoints, "etcd-endpoints", "", "etcd 服务器地址列表，多个地址用逗号分隔（可通过 ETCD_ENDPOINTS 环境变量设置，默认为 http://127.0.0.1:2379）")
	etcdCmd.Flags().StringVar(&etcdEndpoints, "endpoints", "", "etcd 服务器地址列表（已弃用，请使用 --etcd-endpoints）")
	etcdCmd.Flags().MarkDeprecated("endpoints", "请使用 --etcd-endpoints")
	etcdCmd.Flags().StringVar(&etcdCACert, "cacert", "", "CA 证书文件路径（可通过 ETCD_CACERT 环境变量设置，可选）")
	etcdCmd.Flags().StringVar(&etcdCert, "cert", "", "客户端证书文件路径（可通过 ETCD_CERT 环境变量设置，可选）")
	etcdCmd.Flags().StringVar(&etcdKey, "key", "", "客户端私钥文件路径（可通过 ETCD_KEY 环境变量设置，可选）")
	etcdCmd.Flags().StringVar(&etcdUser, "user", "", "etcd 用户名（可通过 ETCD_USER 环境变量设置，可选）")
	etcdCmd.Flags().StringVar(&etcdPassword, "password", "", "etcd 密码（可通过 ETCD_PASSWORD 环境变量设置，可选）")
	etcdCmd.Flags().StringVar(&etcdDialTimeout, "dial-timeout", "", "连接超时时间（可通过 ETCD_DIAL_TIMEOUT 环境变量设置，如 20s，默认 5s）")
	etcdCmd.Flags().StringVar(&etcdCommandTimeout, "command-timeout", "", "命令超时时间（可通过 ETCD_COMMAND_TIMEOUT 环境变量设置，如 60s，默认无超时）")
}

func runEtcdBackup() error {
	// 加载配置（从 .env 文件或环境变量）
	cfg, err := config.LoadConfig(envFile)
	if err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	// 从环境变量获取 etcd 配置（如果命令行参数未设置）
	endpoints := etcdEndpoints
	if endpoints == "" {
		if envEndpoints := os.Getenv("ETCD_ENDPOINTS"); envEndpoints != "" {
			endpoints = envEndpoints
		} else {
			endpoints = "http://127.0.0.1:2379" // 默认地址
		}
	}

	// 解析 endpoints（支持逗号分隔的多个地址）
	var endpointList []string
	if endpoints != "" {
		parts := strings.Split(endpoints, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				endpointList = append(endpointList, part)
			}
		}
	}
	if len(endpointList) == 0 {
		endpointList = []string{"http://127.0.0.1:2379"}
	}

	cacert := etcdCACert
	if cacert == "" {
		cacert = os.Getenv("ETCD_CACERT")
	}

	cert := etcdCert
	if cert == "" {
		cert = os.Getenv("ETCD_CERT")
	}

	key := etcdKey
	if key == "" {
		key = os.Getenv("ETCD_KEY")
	}

	user := etcdUser
	if user == "" {
		user = os.Getenv("ETCD_USER")
	}

	password := etcdPassword
	if password == "" {
		password = os.Getenv("ETCD_PASSWORD")
	}

	dialTimeout := etcdDialTimeout
	if dialTimeout == "" {
		dialTimeout = os.Getenv("ETCD_DIAL_TIMEOUT")
		if dialTimeout == "" {
			dialTimeout = "5s" // 默认 5 秒
		}
	}

	commandTimeout := etcdCommandTimeout
	if commandTimeout == "" {
		commandTimeout = os.Getenv("ETCD_COMMAND_TIMEOUT")
	}

	// 解析超时时间
	dialTimeoutDuration, err := time.ParseDuration(dialTimeout)
	if err != nil {
		return fmt.Errorf("无效的 dial-timeout 格式: %v", err)
	}

	var commandTimeoutDuration time.Duration
	if commandTimeout != "" {
		commandTimeoutDuration, err = time.ParseDuration(commandTimeout)
		if err != nil {
			return fmt.Errorf("无效的 command-timeout 格式: %v", err)
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
	req := controller.EtcdBackupRequest{
		Endpoints:       endpointList,
		CACert:          cacert,
		Cert:            cert,
		Key:             key,
		User:            user,
		Password:        password,
		DialTimeout:     dialTimeoutDuration,
		CommandTimeout:  commandTimeoutDuration,
		CompressMethod:  cfg.CompressMethod,
		OSSEndpoint:     cfg.OSSEndpoint,
		OSSAccessKey:    cfg.OSSAccessKey,
		OSSSecretKey:    cfg.OSSSecretKey,
		OSSBucket:       cfg.OSSBucket,
		OSSObjectPrefix: cfg.OSSObjectPrefix,
	}

	return controller.EtcdBackup(req)
}
