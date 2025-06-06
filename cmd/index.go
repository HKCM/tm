package cmd

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"hash/fnv"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"tm/util"

	"github.com/spf13/cobra"
)

type NoteInfo struct {
	Hcode uint64
	Path  string
	Tags  []string
}

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "更新index",
	Run:   update,
}

func init() {
	rootCmd.AddCommand(indexCmd)
}

type IndexMap struct {
	HashCodeMap map[uint64]string
	TagMap      map[string][]uint64
}

func update(cmd *cobra.Command, args []string) {
	// updateYamlIndex()
	updateIndex()
}

func updateIndex() {
	t := time.Now()
	// 如果有Tag更新则删除缓存
	if util.IsFile(util.Index) {
		err := os.Remove(util.Index)
		if err != nil {
			panic(fmt.Errorf("delete file %s failed,%v", util.Index, err))
		}
	}

	noteRootPath := util.GetNoteRootPath()
	if !util.IsDir(noteRootPath) {
		slog.Error("Target dir not found", "NoteRootPath", noteRootPath)
		return
	}
	slog.Debug("Will update dir tag", "NoteRootPath", noteRootPath)

	var wg sync.WaitGroup
	// indexMap := IndexMap{
	// 	HashCodeMap: make(map[uint64]string),
	// 	TagMap:      make(map[string][]uint64),
	// }

	noteInfoChan := make(chan string, 500)
	// 遍历目录
	err := filepath.Walk(noteRootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}
		if !info.IsDir() {
			wg.Add(1)
			go getNoteTagV2(path, &wg, noteInfoChan)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	go func() {
		wg.Wait()
		close(noteInfoChan) // 关闭 channel
	}()

	var buffer []byte
	for noteInfo := range noteInfoChan {
		slog.Debug(noteInfo)
		buffer = append(buffer, noteInfo...)
		// indexMap.HashCodeMap[noteInfo.Hcode] = noteInfo.Path
		// for _, tag := range noteInfo.Tags {
		// 	if hCodes, ok := indexMap.TagMap[tag]; ok {
		// 		indexMap.TagMap[tag] = append(hCodes, noteInfo.Hcode)
		// 	} else {
		// 		indexMap.TagMap[tag] = []uint64{noteInfo.Hcode}
		// 	}
		// }
	}

	slog.Debug("getTags Write index file")

	// 写入文件
	// saveMapGob(indexMap, util.Index)
	os.WriteFile(util.Index, buffer, 0644)
	util.RemoveUpdateIndexMark()

	slog.Debug("Update index time", "second", time.Since(t))
}

func getNoteTag(filePath string, wg *sync.WaitGroup, noteInfoChan chan NoteInfo) {
	defer wg.Done()
	if !strings.HasSuffix(filePath, ".md") {
		return // 只检查markdown文件
	}

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var tags = []string{}
	standardNote := false
	line := 0
	relativePath, _ := strings.CutPrefix(filePath, util.GetNoteRootPath())

	filepathWithoutEXT := strings.TrimSuffix(relativePath, filepath.Ext(relativePath))     // aaa/bbb.md 获取aaa/bbb
	fileTag := strings.TrimSuffix(filepath.Base(relativePath), filepath.Ext(relativePath)) // aaa/bbb.md 获取bbb
	for scanner.Scan() {
		line++

		if strings.HasPrefix(scanner.Text(), "tags:") {
			standardNote = true
			tagStr, _ := strings.CutPrefix(scanner.Text(), "tags:")
			if strings.TrimSpace(tagStr) != "" {
				// for _, tag := range strings.Split(tagStr, ",") {
				// 	// tags = append(tags, util.FormatTag(tag))
				// 	tags = append(tags, util.FormatTag(tag))
				// }
				tags = append(tags, strings.Split(tagStr, ",")...)
			}
			break
		}
		if line > 1 { // 只需要前两行
			break
		}
	}
	if standardNote {
		hcode := hash(relativePath)
		tags = append(tags, util.FormatTag(fileTag))
		noteInfoChan <- NoteInfo{
			Hcode: hcode,
			Path:  filepathWithoutEXT,
			Tags:  tags,
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func getNoteTagV2(filePath string, wg *sync.WaitGroup, noteInfoChan chan string) {
	defer wg.Done()
	if !strings.HasSuffix(filePath, ".md") {
		return // 只检查markdown文件
	}

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	standardNote := false
	line := 0
	relativePath, _ := strings.CutPrefix(filePath, util.GetNoteRootPath())

	filepathWithoutEXT := strings.TrimSuffix(relativePath, filepath.Ext(relativePath)) // aaa/bbb.md 获取aaa/bbb
	noteInfo := filepathWithoutEXT
	for scanner.Scan() {
		line++

		if strings.HasPrefix(scanner.Text(), "tags:") {
			standardNote = true
			tagStr, _ := strings.CutPrefix(scanner.Text(), "tags:")
			if strings.TrimSpace(tagStr) != "" {
				noteInfo = noteInfo + "," + tagStr
			}
			noteInfo = noteInfo + "\n"
			break
		}
		if line > 1 { // 只需要前两行
			break
		}
	}
	if standardNote {
		noteInfoChan <- noteInfo
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func hash(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// 序列化 Map 到文件
func saveMapGob(indexMap IndexMap, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)

	return encoder.Encode(indexMap)
}

func loadIndexMap(filename string) (IndexMap, error) {
	slog.Debug("loading index")
	var data IndexMap

	file, err := os.Open(filename)
	if err != nil {
		return data, err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&data)

	return data, err
}
func loadIndex(filePath string) (IndexMap, error) {
	slog.Debug("loading index")
	var indexMap = IndexMap{
		HashCodeMap: make(map[uint64]string),
		TagMap:      make(map[string][]uint64),
	}

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		path := strings.SplitN(line, ",", 2)[0]
		hCode := hash(path)
		var tags []string
		idx := strings.LastIndex(line, "/")
		if idx != -1 && idx < len(line)-1 {
			tagStr := line[idx+1:]
			tagStr = util.FormatTag(tagStr)
			tags = strings.Split(tagStr, ",")
		}
		noteInfo := NoteInfo{
			Hcode: hCode,
			Path:  path,
			Tags:  tags,
		}
		indexMap.HashCodeMap[noteInfo.Hcode] = noteInfo.Path
		for _, tag := range noteInfo.Tags {
			if hCodes, ok := indexMap.TagMap[tag]; ok {
				indexMap.TagMap[tag] = append(hCodes, noteInfo.Hcode)
			} else {
				indexMap.TagMap[tag] = []uint64{noteInfo.Hcode}
			}
		}
		//slog.Debug(noteInfo.Tags)

	}

	return indexMap, err
}
