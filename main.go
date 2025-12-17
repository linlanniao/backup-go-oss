/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package main

import (
	_ "backup-to-oss/pkg/version" // 导入版本包以支持构建时注入
	"backup-to-oss/cmd"
)

func main() {
	cmd.Execute()
}
