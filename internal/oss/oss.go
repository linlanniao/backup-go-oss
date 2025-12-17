package oss

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"backup-to-oss/internal/logger"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// Config OSS配置
type Config struct {
	Endpoint     string
	AccessKey    string
	SecretKey    string
	Bucket       string
	ObjectPrefix string
}

// UploadFile 上传文件到OSS
func UploadFile(filePath string, config Config) error {
	// 创建OSS客户端
	client, err := oss.New(config.Endpoint, config.AccessKey, config.SecretKey)
	if err != nil {
		return fmt.Errorf("创建OSS客户端失败: %v", err)
	}

	// 获取存储桶
	bucket, err := client.Bucket(config.Bucket)
	if err != nil {
		return fmt.Errorf("获取存储桶失败: %v", err)
	}

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
	err = bucket.PutObjectFromFile(objectName, filePath)
	if err != nil {
		return fmt.Errorf("上传文件失败: %v", err)
	}

	return nil
}

