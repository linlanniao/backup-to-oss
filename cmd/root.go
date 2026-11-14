/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"backup-to-oss/internal/config"
	"backup-to-oss/internal/logger"

	"github.com/spf13/cobra"
)

var (
	logLevel        string // 日志级别
	logDir          string // 日志目录
	envFile         string // .env 文件路径
	compressMethod  string // 压缩方式
	keepBackupFiles bool   // 是否保留备份文件
	ossEndpoint     string // OSS端点地址
	ossAccessKey    string // OSS AccessKey
	ossSecretKey    string // OSS SecretKey
	ossBucket       string // OSS存储桶名称
	ossObjectPrefix string // OSS对象前缀
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "backup-to-oss",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// 初始化日志器（在 flag 解析后执行）
		level := logLevel
		if level == "" {
			if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
				level = envLevel
			} else {
				level = "info"
			}
		}
		logger.InitLogger(level, logDir)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func loadBackupConfig() (*config.Config, error) {
	cfg, err := config.LoadConfig(envFile)
	if err != nil {
		return nil, err
	}
	if compressMethod != "" {
		cfg.CompressMethod = compressMethod
	}
	switch cfg.CompressMethod {
	case "zstd", "gzip", "none":
		return cfg, nil
	default:
		return nil, fmt.Errorf("不支持的压缩方式 %q（支持 zstd、gzip、none）", cfg.CompressMethod)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.backup-to-oss.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// 添加全局日志级别选项
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "日志级别 (debug/info/warn/error)，可通过 LOG_LEVEL 环境变量设置，默认为 info")
	// 添加日志目录选项
	rootCmd.PersistentFlags().StringVar(&logDir, "log-dir", "", "日志文件输出目录，如果指定则会将日志输出到文件")
	// 添加 .env 文件路径选项
	rootCmd.PersistentFlags().StringVar(&envFile, "env-file", "", ".env 配置文件路径（默认为当前目录下的 .env 文件）")
	// 添加压缩方式选项
	rootCmd.PersistentFlags().StringVarP(&compressMethod, "compress", "c", "", "压缩方式 (zstd/gzip/none)，可通过 COMPRESS_METHOD 环境变量设置，默认为 zstd")
	// 添加保留备份文件选项
	rootCmd.PersistentFlags().BoolVar(&keepBackupFiles, "keep-backup-files", false, "保留备份文件（打包压缩后的文件），不上传到OSS后删除，可通过 KEEP_BACKUP_FILES 环境变量设置")
	// 添加全局 OSS 配置选项
	rootCmd.PersistentFlags().StringVarP(&ossEndpoint, "endpoint", "e", "", "OSS端点地址（可通过 OSS_ENDPOINT 环境变量设置）")
	rootCmd.PersistentFlags().StringVarP(&ossAccessKey, "access-key", "a", "", "OSS AccessKey（可通过 OSS_ACCESS_KEY 环境变量设置）")
	rootCmd.PersistentFlags().StringVarP(&ossSecretKey, "secret-key", "s", "", "OSS SecretKey（可通过 OSS_SECRET_KEY 环境变量设置）")
	rootCmd.PersistentFlags().StringVarP(&ossBucket, "bucket", "b", "", "OSS存储桶名称（可通过 OSS_BUCKET 环境变量设置）")
	rootCmd.PersistentFlags().StringVar(&ossObjectPrefix, "prefix", "", "OSS对象前缀（可通过 OSS_OBJECT_PREFIX 环境变量设置，默认为时间戳）")
}
