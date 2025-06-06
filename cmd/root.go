package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

const (
	EXT    = ".md"
	EDITOR = "code"
	DELIM  = "## "
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "tm",
	Args:  cobra.ArbitraryArgs,
	Short: "note CLI 程序",
	Long:  `这是一个用 Cobra 构建的自建知识库 CLI 程序，用来快速展示曾经做过的笔记。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 获取所有已注册的子命令
		subCmds := map[string]bool{}
		for _, c := range cmd.Commands() {
			subCmds[c.Name()] = true
		}

		// 如果第一个参数是子命令名，按正常流程处理
		if len(args) > 0 && subCmds[args[0]] {
			_ = cmd.Help() // 其实不会进入 Run，此行可选
			return
		}

		// 否则，将 args 传给指定的子命令（如 start）
		cmd.SetArgs(append([]string{"show"}, args...))
		_ = cmd.Execute()

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "开启详细输出")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if verbose {
			handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelDebug,
			})

			logger := slog.New(handler)
			slog.SetDefault(logger)
		}
	}
}
