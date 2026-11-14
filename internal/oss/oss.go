package oss

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"backup-to-oss/internal/logger"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// Config OSS配置
type Config struct {
	Endpoint     string
	Region       string
	AccessKey    string
	SecretKey    string
	Bucket       string
	ObjectPrefix string
}

// regionFromEndpoint 从阿里云公共、内网和双栈端点提取地域。
func regionFromEndpoint(endpoint string) string {
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if region, ok := strings.CutSuffix(host, ".oss.aliyuncs.com"); ok {
		return region
	}
	host, ok := strings.CutSuffix(host, ".aliyuncs.com")
	if !ok || strings.Contains(host, ".") {
		return ""
	}
	region, ok := strings.CutPrefix(host, "oss-")
	region = strings.TrimSuffix(region, "-internal")
	if !ok || region == "accelerate" || region == "accelerate-overseas" {
		return ""
	}
	return region
}

// UploadFile 上传文件到OSS
func UploadFile(filePath string, config Config) error {
	region := config.Region
	if region == "" {
		region = regionFromEndpoint(config.Endpoint)
	}
	if region == "" {
		return fmt.Errorf("无法从 OSS 端点识别地域，请设置 OSS_REGION")
	}
	client := oss.NewClient(oss.LoadDefaultConfig().
		WithEndpoint(config.Endpoint).
		WithRegion(region).
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.AccessKey, config.SecretKey)))

	// 确定对象名称
	objectName := config.ObjectPrefix
	if objectName == "" {
		// 如果没有指定前缀，使用时间戳
		timestamp := time.Now().Format("20060102-150405")
		fileName := filepath.Base(filePath)
		objectName = fmt.Sprintf("backup-%s-%s", timestamp, fileName)
	} else {
		// 如果指定了前缀，添加文件名
		fileName := filepath.Base(filePath)
		// 确保前缀不以 / 开头，但可以以 / 结尾（作为目录分隔符）
		objectName = strings.TrimPrefix(objectName, "/")
		if !strings.HasSuffix(objectName, "/") {
			objectName += "/"
		}
		objectName += fileName
	}

	// 清理对象名称：移除多余的斜杠，确保符合OSS规范
	// OSS对象名称不能以 / 开头，不能包含连续的 //
	objectName = strings.TrimPrefix(objectName, "/")
	objectName = strings.ReplaceAll(objectName, "//", "/")
	// 确保对象名称不为空
	if objectName == "" {
		fileName := filepath.Base(filePath)
		objectName = fileName
	}

	// 打印OSS上传路径信息
	logger.Info("OSS上传路径", "bucket", config.Bucket, "object", objectName, "path", fmt.Sprintf("oss://%s/%s", config.Bucket, objectName))

	// 上传文件
	_, err := client.PutObjectFromFile(context.Background(), &oss.PutObjectRequest{
		Bucket: oss.Ptr(config.Bucket),
		Key:    oss.Ptr(objectName),
	}, filePath)
	if err != nil {
		return fmt.Errorf("上传文件失败: %v", err)
	}

	return nil
}
