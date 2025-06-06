package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"tm/util"

	"github.com/spf13/cobra"
)

const NEW_NOTE_MARK = "(NEW)"

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "新建或编辑笔记",
	Run:   edit,
}

func init() {
	rootCmd.AddCommand(editCmd)
}

func edit(cmd *cobra.Command, args []string) {
	slog.Debug("Sub command edit", "args", args)

	var execCmd *exec.Cmd
	var relativeFilePath string

	fileType, files := findFilePath(args)
	slog.Debug("", "file type", fileType)
	switch fileType {
	case FOLDER:
		util.ExitMessage(fmt.Sprintf("目标是目录,无法编辑 %s ", strings.Join(args, "/")))
	case FILE:
		slog.Debug("目标是文件可以编辑", "args", args)
		relativeFilePath = files[0]
	case NOTHING:
		slog.Debug("目标文件不存在可以新建", "args", args)
		relativeFilePath = strings.Join(args, "/")
	case TAG_FILE, TAG_FILES:
		relativeFilePath = pathSelect(append(files, strings.Join(args, "/")+NEW_NOTE_MARK))
		if strings.HasSuffix(relativeFilePath, NEW_NOTE_MARK) {
			relativeFilePath, _ = strings.CutSuffix(relativeFilePath, NEW_NOTE_MARK)
		}
		slog.Debug("找到TAG文件可以编辑可以新建", "args", args)
	default:
		util.ExitMessage(fmt.Sprintf("无法定位文件 %s ---", strings.Join(args, " ")))
	}

	execCmd = exec.Command(EDITOR, util.GetNoteRootPath()+relativeFilePath+EXT)
	execCmd.Stdout = os.Stdout
	execCmd.Stdin = os.Stdin
	execCmd.Stderr = os.Stderr
	if err := execCmd.Run(); err != nil {
		panic(fmt.Errorf("failed to edit %s: %v", relativeFilePath, err))
	}
	util.SetUpdateIndexMark()
}
