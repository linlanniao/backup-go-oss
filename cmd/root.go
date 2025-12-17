/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"backup-to-oss/internal/logger"

	"github.com/spf13/cobra"
)

var (
	logLevel string // 日志级别
	logDir   string // 日志目录
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
}
