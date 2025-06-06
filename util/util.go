package util

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"strings"
	"unicode/utf8"
)

const (
	RED    = "31"
	GREEN  = "32"
	YELLOW = "33"
	BLUE   = "34"
)

type Config struct {
	Path string
}

var COMMON = []string{"#", "//", "-- "}

var (
	UserHome            string
	ConfigPath          string // ~/.tm/config
	Index               string // 唯一的索引文件, ~/note/.index
	NoteRootPath        string
	IndexMarkUpdateMark string
)

func init() {
	user, _ := user.Current()
	UserHome = user.HomeDir
	ConfigPath = UserHome + "/.tm/config"
	NoteRootPath = getConfig(ConfigPath).Path
	Index = UserHome + "/.tm/.index"
	IndexMarkUpdateMark = UserHome + "/.tm/updateIndex"

}

func colorPrintln(color, text string) {
	fmt.Printf("\x1b[%sm%s\x1b[0m\n", color, text)
}

func colorPrint(text string) {
	for _, s := range COMMON {
		n := strings.Index(text, s)
		if n != -1 {
			fmt.Printf("\x1b[%sm%s\x1b[0m", YELLOW, text[:n])
			fmt.Printf("\x1b[%sm%s\x1b[0m", GREEN, text[n:])
			fmt.Println()
			return
		}
	}
	colorPrintln(YELLOW, text)
}

func ColorfulPrint(text string) {
	for _, t := range strings.Split(text, "\n") {
		// 排除以```开头的行
		if strings.HasPrefix(t, "```") {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(t), "#") || strings.HasPrefix(strings.TrimSpace(t), "//") {
			colorPrintln(GREEN, t)
			continue
		}
		colorPrint(t)
	}
}

func GetUpdateIndexMark() bool {
	return IsFile(IndexMarkUpdateMark)
}

func SetUpdateIndexMark() {
	if !IsFile(IndexMarkUpdateMark) {
		os.Create(IndexMarkUpdateMark)
	}
}

func RemoveUpdateIndexMark() {
	if IsFile(IndexMarkUpdateMark) {
		os.Remove(IndexMarkUpdateMark)
	}
}

func calculateWidth(str string) int {
	width := 0
	for _, r := range str {
		if utf8.RuneLen(r) == 3 { // 如果字符长度为 3，则为中文字符
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

func TablePrint(list []string, column int) {
	if len(list) == 0 {
		slog.Debug("No list")
		return
	}
	if len(list) < column {
		column = len(list)
	}

	w := make([]int, column)
	for i, s := range list {
		// 计算每列的宽度
		l := calculateWidth(s) + 1
		if l >= w[i%column] {
			w[i%column] = l
		}
	}
	slog.Debug("ListTablePrint", "w", w)
	a := len(list) % column
	if a != 0 {
		for i := 0; i < column-a; i++ {
			list = append(list, "")
		}
	}
	printTableLine(w)
	for i := 0; i < len(list); i += column {
		for j := i; j < i+column && j < len(list); j++ {
			fmt.Printf("|%s%s%-*s", " ", list[j], w[j%column]-calculateWidth(list[j]), " ")
		}
		fmt.Printf("|\n")
	}
	printTableLine(w)
}

func printTableLine(columnWidth []int) {
	for _, l := range columnWidth {
		fmt.Printf("+")
		fmt.Printf("%s-", strings.Repeat("-", l))
	}
	fmt.Printf("+\n")
}

// ------------------------------以下为工具函数----------------------------
// 判断所给路径是否为文件夹
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		//  no such file or directory
		return false
	}
	return s.IsDir()
}

// 判断所给路径是否为文件
func IsFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !s.IsDir()
}

func ExitMessage(msg string) {
	fmt.Println(msg)
	os.Exit(0)
}

func GetNoteRootPath() string {
	return NoteRootPath
}

func getConfig(configPath string) (config Config) {
	inputFile, err := os.Open(configPath)
	if err != nil {
		panic(err)
	}
	defer inputFile.Close()
	scanner := bufio.NewScanner(inputFile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "path=") {
			config.Path = strings.Split(line, "=")[1]
		}
	}
	return
}

func GetNotePart(filePath, part string) string {

	file, err := os.Open(filePath)
	if err != nil {
		panic(fmt.Errorf("failed to read file: %s, %v", filePath, err))
	}
	defer file.Close()

	var context strings.Builder

	found := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, part) {
			if found {
				break
			}
			found = true
			context.Reset()
		} else {
			context.WriteString(line)
			context.WriteString("\n")
		}
	}

	return context.String()

}

// Lowercaseable 是一个接口约束，只允许 string 或 []string 类型
type Lowercaseable interface {
	string | []string
}

func FormatTag[T Lowercaseable](input T) T {
	replacer := strings.NewReplacer(" ", "", "_", "")
	var result any
	switch v := any(input).(type) {
	case string:
		result = strings.ToLower(replacer.Replace(v))
	case []string:
		lowered := make([]string, len(v))
		for i, s := range v {
			lowered[i] = strings.ToLower(replacer.Replace(s))
		}
		result = lowered
	default:
		// 编译时应该不会走到这里，因为 T 被类型约束了
		panic("unsupported type")
	}

	return result.(T)

}
