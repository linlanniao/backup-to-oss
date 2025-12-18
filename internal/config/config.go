package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config 应用配置
type Config struct {
	DirPaths        []string // 支持多个目录
	FilePaths       []string // 支持多个文件
	ExcludePatterns []string // 排除模式列表
	CompressMethod  string   // 压缩方式 (zstd/gzip/none)
	OSSEndpoint     string
	OSSAccessKey    string
	OSSSecretKey    string
	OSSBucket       string
	OSSObjectPrefix string
}

// LoadConfig 加载配置，优先从命令行参数，其次从环境变量，最后从 .env 文件
// envFile 参数指定 .env 文件路径，如果为空则使用默认路径（当前目录下的 .env）
func LoadConfig(envFile string) (*Config, error) {
	// 尝试加载 .env 文件（如果存在）
	if envFile != "" {
		// 如果指定了 .env 文件路径，使用指定的路径
		if err := godotenv.Load(envFile); err != nil {
			// 如果文件不存在，不报错，继续使用环境变量
			_ = err
		}
	} else {
		// 如果没有指定路径，尝试加载当前目录下的 .env 文件
		_ = godotenv.Load()
	}

	// 解析多个目录（逗号分隔）
	dirsStr := getEnvOrDefault("DIRS_TO_BACKUP", "")
	var dirPaths []string
	if dirsStr != "" {
		dirs := strings.Split(dirsStr, ",")
		for _, dir := range dirs {
			dir = strings.TrimSpace(dir)
			if dir != "" {
				dirPaths = append(dirPaths, dir)
			}
		}
	}

	// 解析多个文件（逗号分隔）
	filesStr := getEnvOrDefault("FILES_TO_BACKUP", "")
	var filePaths []string
	if filesStr != "" {
		files := strings.Split(filesStr, ",")
		for _, file := range files {
			file = strings.TrimSpace(file)
			if file != "" {
				filePaths = append(filePaths, file)
			}
		}
	}

	// 解析排除模式（逗号分隔）
	excludeStr := getEnvOrDefault("EXCLUDE_PATTERNS", "")
	var excludePatterns []string
	if excludeStr != "" {
		patterns := strings.Split(excludeStr, ",")
		for _, pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if pattern != "" {
				excludePatterns = append(excludePatterns, pattern)
			}
		}
	}

	// 获取压缩方式，默认为 zstd
	compressMethod := getEnvOrDefault("COMPRESS_METHOD", "zstd")

	cfg := &Config{
		DirPaths:        dirPaths,
		FilePaths:       filePaths,
		ExcludePatterns: excludePatterns,
		CompressMethod:  compressMethod,
		OSSEndpoint:     getEnvOrDefault("OSS_ENDPOINT", ""),
		OSSAccessKey:    getEnvOrDefault("OSS_ACCESS_KEY", ""),
		OSSSecretKey:    getEnvOrDefault("OSS_SECRET_KEY", ""),
		OSSBucket:       getEnvOrDefault("OSS_BUCKET", ""),
		OSSObjectPrefix: getEnvOrDefault("OSS_OBJECT_PREFIX", ""),
	}

	return cfg, nil
}

// MergeWithFlags 将命令行参数合并到配置中（命令行参数优先级更高）
func (c *Config) MergeWithFlags(dirPath, excludePatterns, compressMethod, endpoint, accessKey, secretKey, bucket, prefix string) {
	if dirPath != "" {
		// 如果命令行指定了目录，解析逗号分隔的多个目录
		dirs := strings.Split(dirPath, ",")
		var dirPaths []string
		for _, dir := range dirs {
			dir = strings.TrimSpace(dir)
			if dir != "" {
				dirPaths = append(dirPaths, dir)
			}
		}
		if len(dirPaths) > 0 {
			c.DirPaths = dirPaths
		}
	}
	if excludePatterns != "" {
		// 解析排除模式
		patterns := strings.Split(excludePatterns, ",")
		var excludeList []string
		for _, pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if pattern != "" {
				excludeList = append(excludeList, pattern)
			}
		}
		if len(excludeList) > 0 {
			c.ExcludePatterns = excludeList
		}
	}
	if compressMethod != "" {
		c.CompressMethod = compressMethod
	}
	if endpoint != "" {
		c.OSSEndpoint = endpoint
	}
	if accessKey != "" {
		c.OSSAccessKey = accessKey
	}
	if secretKey != "" {
		c.OSSSecretKey = secretKey
	}
	if bucket != "" {
		c.OSSBucket = bucket
	}
	if prefix != "" {
		c.OSSObjectPrefix = prefix
	}
}

// MergeWithFileFlags 将命令行参数合并到配置中（用于文件备份，命令行参数优先级更高）
func (c *Config) MergeWithFileFlags(filePath, compressMethod, endpoint, accessKey, secretKey, bucket, prefix string) {
	if filePath != "" {
		// 如果命令行指定了文件，解析逗号分隔的多个文件
		files := strings.Split(filePath, ",")
		var filePaths []string
		for _, file := range files {
			file = strings.TrimSpace(file)
			if file != "" {
				filePaths = append(filePaths, file)
			}
		}
		if len(filePaths) > 0 {
			c.FilePaths = filePaths
		}
	}
	if compressMethod != "" {
		c.CompressMethod = compressMethod
	}
	if endpoint != "" {
		c.OSSEndpoint = endpoint
	}
	if accessKey != "" {
		c.OSSAccessKey = accessKey
	}
	if secretKey != "" {
		c.OSSSecretKey = secretKey
	}
	if bucket != "" {
		c.OSSBucket = bucket
	}
	if prefix != "" {
		c.OSSObjectPrefix = prefix
	}
}

// Validate 验证配置是否完整（用于目录备份）
func (c *Config) Validate() error {
	if len(c.DirPaths) == 0 {
		return fmt.Errorf("目录路径未设置（通过 --path 参数或 DIRS_TO_BACKUP 环境变量，支持多个目录用逗号分隔）")
	}
	if c.OSSEndpoint == "" {
		return fmt.Errorf("OSS端点未设置（通过 --endpoint 参数或 OSS_ENDPOINT 环境变量）")
	}
	if c.OSSAccessKey == "" {
		return fmt.Errorf("OSS AccessKey未设置（通过 --access-key 参数或 OSS_ACCESS_KEY 环境变量）")
	}
	if c.OSSSecretKey == "" {
		return fmt.Errorf("OSS SecretKey未设置（通过 --secret-key 参数或 OSS_SECRET_KEY 环境变量）")
	}
	if c.OSSBucket == "" {
		return fmt.Errorf("OSS存储桶未设置（通过 --bucket 参数或 OSS_BUCKET 环境变量）")
	}
	return nil
}

// ValidateFileConfig 验证文件备份配置是否完整
func (c *Config) ValidateFileConfig() error {
	if len(c.FilePaths) == 0 {
		return fmt.Errorf("文件路径未设置（通过 --path 参数或 FILES_TO_BACKUP 环境变量，支持多个文件用逗号分隔）")
	}
	if c.OSSEndpoint == "" {
		return fmt.Errorf("OSS端点未设置（通过 --endpoint 参数或 OSS_ENDPOINT 环境变量）")
	}
	if c.OSSAccessKey == "" {
		return fmt.Errorf("OSS AccessKey未设置（通过 --access-key 参数或 OSS_ACCESS_KEY 环境变量）")
	}
	if c.OSSSecretKey == "" {
		return fmt.Errorf("OSS SecretKey未设置（通过 --secret-key 参数或 OSS_SECRET_KEY 环境变量）")
	}
	if c.OSSBucket == "" {
		return fmt.Errorf("OSS存储桶未设置（通过 --bucket 参数或 OSS_BUCKET 环境变量）")
	}
	return nil
}

// getEnvOrDefault 获取环境变量，如果不存在则返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
