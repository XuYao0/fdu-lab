package editor

import (
	"fmt"
	"lab1/common"
	"strings"
)

type TextEditorInterface interface {
	common.Editor
	Show(startLine, endLine int)
	Append(content string)
	Insert(line, col int, text string)
	Delete(line, col, length int)
	Replace(line, col, length int, text string)
}

// TextEditor 文本编辑器（具体组件）
type TextEditor struct {
	//editorType string
	filePath     string
	lines        []string
	isModified   bool
	undoStack    []Command
	redoStack    []Command
	logEnabled   bool
	workspaceApi common.WorkSpaceApi
	//observers  []workspace.Observer // 观察者列表（可选，用于编辑器级事件）
}

// RegisterObsever()

// IsLogEnabled 实现日志状态方法
func (t *TextEditor) IsLogEnabled() bool {
	return t.logEnabled
}

// //这里要加上对文件首行的更新
// func (t *TextEditor) SetLogEnabled(enabled bool) {
//     t.logEnabled = enabled
// 	if enabled{

// 	}else{

//		}
//	}
//
// SetLogEnabled 设置日志开关，并在内存中更新文件首行的# log标记（不直接持久化到磁盘）
func (t *TextEditor) SetLogEnabled(enabled bool) {
	// 1. 记录旧状态，若状态无变化则直接返回，避免无效操作
	oldEnabled := t.logEnabled
	t.logEnabled = enabled
	if oldEnabled == enabled {
		return
	}

	// 2. 根据开关状态，在内存中处理首行的# log标记
	if enabled {
		// 开启日志：首行无# log则插入（仅内存中）
		t.addLogMarkerInMemory()
	} else {
		// 关闭日志：首行有# log则移除（仅内存中）
		t.removeLogMarkerInMemory()
	}
}

// addLogMarkerInMemory 仅在内存中给文件首行添加# log标记（无则加）
func (t *TextEditor) addLogMarkerInMemory() {
	if len(t.lines) == 0 {
		t.lines = []string{"# log"}
	} else {

		firstLine := strings.TrimSpace(t.lines[0])
		if firstLine != "# log" {
			t.lines = append([]string{"# log"}, t.lines...)
		}
	}
	// 标记文件为已修改（供后续持久化逻辑判断）
	t.MarkAsModified(true)
}

// removeLogMarkerInMemory 仅在内存中移除文件首行的# log标记（有则删）
func (t *TextEditor) removeLogMarkerInMemory() {
	if len(t.lines) == 0 {
		return
	}

	// 去除首行空格后检查是否是目标标记
	firstLine := strings.TrimSpace(t.lines[0])
	if firstLine == "# log" {
		t.lines = t.lines[1:]
		t.MarkAsModified(true)
	}
}

// NewTextEditor 创建文本编辑器实例
func NewTextEditor(filePath, content string, wsApi common.WorkSpaceApi) *TextEditor {
	return &TextEditor{
		filePath:     filePath,
		lines:        strings.Split(content, "\n"),
		workspaceApi: wsApi,
		//observers: make([]workspace.Observer, 0),
	}
}

// GetFilePath 获取文件路径
func (te *TextEditor) GetFilePath() string {
	return te.filePath
}

// IsModified 检查是否修改
func (te *TextEditor) IsModified() bool {
	return te.isModified
}

// MarkAsModified 标记修改状态
func (te *TextEditor) MarkAsModified(modified bool) {
	te.isModified = modified
}

// ExecuteCommand 执行命令（命令模式入口）
func (te *TextEditor) ExecuteCommand(command Command) {
	command.Execute()
	te.undoStack = append(te.undoStack, command)
	te.redoStack = nil // 新操作清空重做栈
	te.isModified = true
}

// Undo 撤销操作
func (te *TextEditor) Undo() error {
	if len(te.undoStack) == 0 {
		return nil
	}
	cmd := te.undoStack[len(te.undoStack)-1]
	cmd.Undo()
	te.undoStack = te.undoStack[:len(te.undoStack)-1]
	te.redoStack = append(te.redoStack, cmd)
	return nil
}

// Redo 重做操作
func (te *TextEditor) Redo() error {
	if len(te.redoStack) == 0 {
		fmt.Println("redo stack is empty!")
		return nil
	}
	cmd := te.redoStack[len(te.redoStack)-1]
	cmd.Execute()
	te.redoStack = te.redoStack[:len(te.redoStack)-1]
	te.undoStack = append(te.undoStack, cmd)
	return nil
}

// GetContent 获取完整内容（供保存）
func (te *TextEditor) GetContent() string {
	return strings.Join(te.lines, "\n")
}

func (te *TextEditor) getLine(lineNum int) (string, bool) {
	if lineNum < 0 || lineNum >= len(te.lines) {
		return "", false
	}
	return te.lines[lineNum], true
}

func (te *TextEditor) setLine(lineNum int, content string) bool {
	if lineNum < 0 || lineNum >= len(te.lines) {
		return false
	}
	te.lines[lineNum] = content
	return true
}

func (te *TextEditor) insertLine(lineNum int, content string) bool {
	if lineNum < 0 || lineNum > len(te.lines) {
		return false
	}
	te.lines = append(te.lines[:lineNum], append([]string{content}, te.lines[lineNum:]...)...)
	return true
}

func (te *TextEditor) deleteLine(lineNum int) (string, bool) {
	if lineNum < 0 || lineNum >= len(te.lines) {
		return "", false
	}
	deleted := te.lines[lineNum]
	te.lines = append(te.lines[:lineNum], te.lines[lineNum+1:]...)
	return deleted, true
}
