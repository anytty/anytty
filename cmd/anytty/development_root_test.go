package main

import "github.com/spf13/cobra"

// newDevelopmentRootCmd 只为历史 smoke/harness 测试装配 v3 命令。
// 产品 newRootCmd 不调用它，因此 release help 和命令解析都无法进入这些开发入口。
func newDevelopmentRootCmd() *cobra.Command {
	var socket, logFile, configPath string
	command := &cobra.Command{Use: "anytty-test"}
	command.PersistentFlags().StringVar(&socket, "socket", "", "test socket path")
	command.PersistentFlags().StringVar(&logFile, "log-file", "", "test log path")
	command.PersistentFlags().StringVar(&configPath, "config", "", "test config path")
	command.AddCommand(v3Command(&socket, &logFile, &configPath))
	return command
}
