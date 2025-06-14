package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"tm/util"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

type FileType string

const (
	FOLDER    = "FOLDER"
	FILE      = "FILE"
	TAG_FILE  = "TAG_FILE"
	TAG_FILES = "TAG_FILES"
	NOTHING   = "NOTHING"
)

const MAX_LINE = 30

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "显示笔记",
	Run:   show,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func show(cmd *cobra.Command, args []string) {
	if util.GetUpdateIndexMark() {
		slog.Debug("It has update index mark,should update index")
		updateIndex()
	}

	slog.Debug("cmdShow origin", "Args", args)
	args = util.FormatTag(args)

	fileType, files := findFilePath(args)
	slog.Debug("", "fileType", fileType, "files", files)
	switch fileType {
	case FOLDER:
		showNoteFolder(files[0])
	case FILE, TAG_FILE, TAG_FILES:
		showNoteFile(files, DELIM)
	default:
		util.ExitMessage(fmt.Sprintf("未发现相关文件, %s ", strings.Join(args, " ")))
	}
}

func showNoteFile(noteFiles []string, part string) {
	noteFile := noteFiles[0]
	if len(noteFiles) > 1 {
		noteFile = pathSelect(noteFiles)
	}

	filePath := util.GetNoteRootPath() + noteFile + EXT
	if !util.IsFile(filePath) {
		// 从index显示存在但是实际显示不存在,则说明可能是手动删除了,所以也需要更新index
		updateIndex()
		// 然后再次检测
		fmt.Println("文件不存在,请重试")
		os.Exit(1)
	}
	text := util.GetNotePart(filePath, part)
	if len(strings.Split(text, "\n")) < MAX_LINE {
		util.ColorfulPrint(text)
	} else {
		cmd := exec.Command(EDITOR, filePath)
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			panic(fmt.Errorf("failed to edit %s: %v", filePath, err))
		}
	}
}

func showNoteFolder(dirPath string) {
	abDirPath := util.GetNoteRootPath() + dirPath

	if !util.IsDir(abDirPath) {
		panic(fmt.Errorf("note  %s is not existed", abDirPath))
	}

	cmd := exec.Command(EDITOR, abDirPath)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("failed to edit %s: %v", abDirPath, err))
	}

}

// findFilePath 查找文件路径（支持模糊）
func findFilePath(args []string) (fileType FileType, files []string) {
	// 什么都没有
	if len(args) == 0 {
		fileType = NOTHING
		return
	}

	l := len(args)
	folders := args[:l-1]
	tag := args[l-1]
	rootNatePath := util.GetNoteRootPath()
	prefixPath := strings.Join(folders, "/")

	// 如果输入的目录是错误的 也就不用往下执行了
	if !util.IsDir(rootNatePath + prefixPath) {
		slog.Debug("目录不存在", "prefixPath", prefixPath)
		fileType = NOTHING
		return
	}
	tmp_path := strings.Join(args, "/")

	// 尝试匹配具体文件
	if util.IsFile(rootNatePath + tmp_path + EXT) {
		fileType = FILE
		files = []string{tmp_path}
		return
	}
	slog.Debug("未能精准匹配文件")

	// 尝试匹配具体目录
	if util.IsDir(rootNatePath + tmp_path) {
		fileType = FOLDER
		files = []string{tmp_path}
		return
	}
	slog.Debug("未能匹配目录")

	/*
		剩下tag查询(精准tag查询和模糊tag查询)和不存在
	*/

	// indexMap, err := loadIndexMap(util.Index)
	indexMap, err := loadIndex(util.Index)
	if err != nil {
		panic(err)
	}

	exists := make(map[uint64]struct{}) // 用来标记是否存在重复添加

	// 全局精准匹配
	if hCodes, ok := indexMap.TagMap[tag]; ok {
		fileType = TAG_FILES
		slog.Debug("精准匹配到Tag", "tag", tag)
		for _, hcode := range hCodes {
			file := indexMap.HashCodeMap[hcode]
			if addToFiles(prefixPath, hcode, indexMap.HashCodeMap, exists) {
				files = append(files, file)
			}
		}
		switch len(files) {
		case 0:
			fileType = NOTHING
		case 1:
			fileType = TAG_FILE
		default:
			fileType = TAG_FILES
		}
		slog.Debug("精准匹配Tag", "target", tag, "totalFilepaths", files)
		return
	}
	slog.Debug("未能精准匹配目标Tag, 进行模糊匹配", "target tag", tag)

	// 全局模糊匹配
	for k, hCodes := range indexMap.TagMap {
		slog.Debug(k)
		if strings.Contains(k, tag) {
			for _, hcode := range hCodes {
				if addToFiles(prefixPath, hcode, indexMap.HashCodeMap, exists) {
					files = append(files, indexMap.HashCodeMap[hcode])
				}
			}
		}
	}
	slog.Debug("模糊匹配Tag", "target", tag, "totalFilepaths", files)
	switch len(files) {
	case 0:
		fileType = NOTHING
	case 1:
		fileType = TAG_FILE
	default:
		fileType = TAG_FILES
	}

	return
}

func addToFiles(prefix string, hcode uint64, hashCodeMap map[uint64]string, exists map[uint64]struct{}) bool {
	tmp_path := hashCodeMap[hcode]
	if strings.HasPrefix(tmp_path, prefix) {
		if _, ok := exists[hcode]; ok {
			return false
		}
		exists[hcode] = struct{}{}
		return true
	}
	return false
}

func pathSelect(filePaths []string) (result string) {
	filePaths = append(filePaths, "Cancel")
	prompt := promptui.Select{
		Label: "Select a Note",
		Items: filePaths,
	}

	index, result, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	if result == "Cancel" {
		os.Exit(0)
	}
	slog.Debug(fmt.Sprintf("You selected %d: %s\n", index+1, result))
	return
}
