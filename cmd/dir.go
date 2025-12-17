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
	dirPath         string
	excludePatterns string
	ossEndpoint     string
	ossAccessKey    string
	ossSecretKey    string
	ossBucket       string
	ossObjectPrefix string
)

// dirCmd represents the dir command
var dirCmd = &cobra.Command{
	Use:   "dir",
	Short: "压缩备份整个目录到OSS",
	Long: `将指定目录压缩为gzip格式后上传到阿里云OSS。

配置可以通过以下方式提供：
1. .env 文件（推荐）
2. 环境变量
3. 命令行参数（优先级最高）

示例:
  backup-to-oss dir --path /path/to/dir
  或
  backup-to-oss dir --path /path/to/dir --endpoint oss-cn-hangzhou.aliyuncs.com --bucket my-bucket`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDirBackup(); err != nil {
			logger.Error("备份失败", "error", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(dirCmd)

	dirCmd.Flags().StringVarP(&dirPath, "path", "p", "", "要备份的目录路径，支持多个目录用逗号分隔（可通过 DIRS_TO_BACKUP 环境变量设置）")
	dirCmd.Flags().StringVarP(&excludePatterns, "exclude", "x", "", "排除模式，支持多个模式用逗号分隔（可通过 EXCLUDE_PATTERNS 环境变量设置），支持 glob 模式，如: *.log,node_modules,.git")
	dirCmd.Flags().StringVarP(&ossEndpoint, "endpoint", "e", "", "OSS端点地址（可通过 OSS_ENDPOINT 环境变量设置）")
	dirCmd.Flags().StringVarP(&ossAccessKey, "access-key", "a", "", "OSS AccessKey（可通过 OSS_ACCESS_KEY 环境变量设置）")
	dirCmd.Flags().StringVarP(&ossSecretKey, "secret-key", "s", "", "OSS SecretKey（可通过 OSS_SECRET_KEY 环境变量设置）")
	dirCmd.Flags().StringVarP(&ossBucket, "bucket", "b", "", "OSS存储桶名称（可通过 OSS_BUCKET 环境变量设置）")
	dirCmd.Flags().StringVarP(&ossObjectPrefix, "prefix", "", "", "OSS对象前缀（可通过 OSS_OBJECT_PREFIX 环境变量设置，默认为时间戳）")
}

func runDirBackup() error {
	// 加载配置（从 .env 文件或环境变量）
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	// 合并命令行参数（命令行参数优先级更高）
	cfg.MergeWithFlags(dirPath, excludePatterns, ossEndpoint, ossAccessKey, ossSecretKey, ossBucket, ossObjectPrefix)

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return err
	}

	// 构建请求
	req := controller.DirBackupRequest{
		DirPaths:        cfg.DirPaths,
		ExcludePatterns: cfg.ExcludePatterns,
		OSSEndpoint:     cfg.OSSEndpoint,
		OSSAccessKey:    cfg.OSSAccessKey,
		OSSSecretKey:    cfg.OSSSecretKey,
		OSSBucket:       cfg.OSSBucket,
		OSSObjectPrefix: cfg.OSSObjectPrefix,
	}

	return controller.DirBackup(req)
}
