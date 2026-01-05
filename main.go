package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"lab1/TreeAdapter"
	"lab1/common"
	"lab1/editor"
	"lab1/log"
	"lab1/statistics"
	"lab1/storage"
	"lab1/workspace"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>
// 计时器绑定
var timeStatistics = &statistics.Statistics{}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
func main() {
	// 1. 初始化依赖组件
	fileStorage := storage.NewLocalStorage("./workspace_state.json") // 状态存储路径
	logModule := log.NewLogModule()

	// 2. 初始化工作区
	ws := workspace.NewWorkspace("./workspace_state.json")

	// 3. 日志模块订阅工作区事件（观察者模式）
	ws.RegisterObserver(logModule)

	//计时器绑定
	timeStatistics = statistics.NewStatistics()
	ws.RegisterObserver(timeStatistics)
	//timeStatistics.GetFormattedDuration()
	//日志模块订阅编辑器事件

	// 4. 从本地存储恢复上次工作区状态（备忘录模式）
	if err := restoreWorkspaceState(ws, fileStorage); err != nil {
		fmt.Printf("恢复工作区失败，使用新状态: %v\n", err)
	} else {
		fmt.Println("工作区已恢复上次状态")
	}

	content := readFile("t.txt")
	err := editor.SpellCheckTxt(content)
	if err != nil {
		fmt.Println(err)
	}

	// 5. 启动交互循环，处理用户指令
	startInteractiveLoop(ws)
}

// 修复后的 restoreWorkspaceState 函数
func restoreWorkspaceState(ws *workspace.Workspace, storage *storage.LocalStorage) error {
	// 调用 Workspace 的 RestoreState 方法，传入编辑器工厂函数
	// 工厂函数复用之前定义的 editor.EditorFactory（需确保已导入 editor 包）
	fmt.Println("restoreWorkspaceState")
	return ws.RestoreState(editor.EditorFactory)
}

// 启动用户交互循环
func startInteractiveLoop(ws *workspace.Workspace) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("编辑器启动完成，支持指令: load/save/close/undo/exit....")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		handleCommand(ws, input, true)
		//fmt.Printf("[debug]active_file: %s\n", ws.GetActiveEditor().GetFilePath())
		activeEditor := ws.GetActiveEditor()
		if activeEditor == nil {
			fmt.Println("[debug]active_file: 无激活的编辑器/文件")
		} else {
			fmt.Printf("[debug]active_file: %s\n", activeEditor.GetFilePath())
		}
	}
}

// 处理用户指令
func handleCommand(ws *workspace.Workspace, input string, debug bool) {
	input = strings.TrimSpace(input)
	parts := strings.SplitN(input, " ", 10)
	if len(parts) == 0 {
		fmt.Println("无效指令")
		return
	}
	cmd := parts[0]
	switch cmd {
	case "load": //完成
		_load(ws, parts, false)
	case "save": //完成
		_Save(ws, input, debug, parts)
	case "close": //完成
		_close(ws, parts)
	case "init": //完成
		_init(ws, input)
	case "undo":
		_undo(ws)
	case "redo":
		_redo(ws)
	case "editor-list": //完成
		_EditorList(ws)
	case "edit": //完成
		_edit(ws, parts)
	case "exit":
		_exit(ws)
	case "dir-tree": //完成
		//_dirTree(ws, parts)
		_dirTreeV2(ws, parts)
	case "append":
		_append(ws, parts)
	case "insert":
		_insert(ws, parts)
	case "show":
		_show(ws, parts)
	case "delete":
		// 先尝试按XML指令处理，若不是XML编辑器，再按文本指令处理
		if !_xmlDelete(ws, parts) {
			_delete(ws, parts)
		}
	case "replace":
		_replace(ws, parts)
	case "log-on":
		_LogOn(ws, parts)
	case "log-off":
		_LogOff(ws, parts)
	case "log-show":
		_LogShow(ws, parts)
	//case :
	case "insert-before":
		_insertBefore(ws, parts)
	case "append-child":
		_appendChild(ws, parts)
	case "edit-id":
		_editId(ws, parts)
	case "edit-text":
		_editText(ws, parts)
	case "xml-tree":
		//_xmlTree(ws, parts)
		_xmlTreeV2(ws, parts)
	case "spell-check":
		_spellCheck(ws, parts)
	default:
		fmt.Println("未知指令，支持: load/save/close/undo/exit")
	}
}
func _spellCheck(ws *workspace.Workspace, parts []string) {
	if len(parts) < 2 {
		_ed := ws.GetActiveEditor()
		if _ed == nil {
			fmt.Println("当前没有活动文件")
			return
		}
		content := readFile(_ed.GetFilePath())
		ext := strings.ToLower(filepath.Ext(_ed.GetFilePath()))
		var err error
		switch ext {
		case ".txt":
			err = editor.SpellCheckTxt(content)
		case ".xml":
			err = editor.SpellCheckXML(content)
		default:
			fmt.Printf("不支持的文件类型: %s（仅支持 .txt/.xml）\n", ext)
			return
		}
		if err != nil {
			fmt.Println("拼写检查错误:", err)
		}
		return
	}

	parts = parts[1:]
	part := strings.TrimSpace(strings.Join(parts, ""))
	filePath := "files\\" + part
	if _, ok := ws.OpenEditors[filePath]; !ok {
		fmt.Println("目标文件未在工作区打开")
		return
	}
	content := readFile(filePath)
	ext := strings.ToLower(filepath.Ext(filePath))
	var err error
	switch ext {
	case ".txt":
		err = editor.SpellCheckTxt(content)
	case ".xml":
		err = editor.SpellCheckXML(content)
	default:
		fmt.Printf("不支持的文件类型: %s（仅支持 .txt/.xml）\n", ext)
		return
	}
	if err != nil {
		fmt.Println("拼写检查错误:", err)
	}
	return
}
func _load(ws *workspace.Workspace, parts []string, debug bool) {
	if len(parts) < 2 {
		fmt.Println("请指定文件路径: load [path]")
		return
	}
	_editor, err := ws.LoadFile(parts[1], editor.EditorFactory)
	if err != nil {
		fmt.Printf("加载失败: %v\n", err)
	} else {
		fmt.Printf("已加载文件: %s（%s）\n",
			_editor.GetFilePath(),
			map[bool]string{true: "已修改", false: "未修改"}[_editor.IsModified()])
		if debug {
			fmt.Println("[debug] 当前活动文件是" + ws.GetActiveEditor().GetFilePath())
		}
	}
}

func _close(ws *workspace.Workspace, parts []string) {
	if len(parts) < 2 {
		fmt.Println("请指定文件路径: close [path]")
		return
	}
	if err := ws.CloseFile(parts[1]); err != nil {
		fmt.Printf("关闭失败: %v\n", err)
	} else {
		fmt.Printf("已关闭文件: %s\n", parts[1])
	}
}

func _undo(ws *workspace.Workspace) {
	if err := ws.GetActiveEditor().Undo(); err != nil {
		fmt.Printf("undo失败: %v\n", err)
	} else {
		fmt.Println("undo成功")
	}

}

func _redo(ws *workspace.Workspace) {
	if err := ws.GetActiveEditor().Redo(); err != nil {
		fmt.Printf("redo失败: %v\n", err)
	} else {
		fmt.Println("redo成功")
	}
}

func _exit(ws *workspace.Workspace) {
	// 退出前保存工作区状态
	memento := ws.CreateMemento()
	if err := storage.NewLocalStorage("./workspace_state.json").SaveMemento(memento); err != nil {
		fmt.Printf("保存工作区状态失败: %v\n", err)
	}
	fmt.Println("程序退出")
	os.Exit(0)
}

func _dirTree(ws *workspace.Workspace, parts []string) {
	// 确定目标目录（默认当前工作目录）
	targetDir := "."
	if len(parts) >= 2 {
		targetDir = parts[1]
	}

	// 验证目录是否存在
	if _, err := os.Stat(targetDir); err != nil {
		fmt.Printf("目录不存在: %v\n", err)
		return
	}

	// 生成并打印目录树
	tree, err := generateDirectoryTree(targetDir)
	if err != nil {
		fmt.Printf("生成目录树失败: %v\n", err)
		return
	}
	fmt.Print(tree)
}
func _dirTreeV2(ws *workspace.Workspace, parts []string) {
	// 默认当前目录，清理冗余路径
	targetDir := "."
	if len(parts) >= 2 {
		targetDir = filepath.Clean(parts[1]) // 清理路径，跨平台更友好
	}

	//  存在 + 是目录
	stat, err := os.Stat(targetDir)
	if err != nil {

		fmt.Printf("访问路径失败: %v\n", err)
		return
	}
	if !stat.IsDir() {
		fmt.Printf("指定路径不是目录: %s\n", targetDir)
		return
	}

	dirTreeAdapter := &TreeAdapter.FileTreeAdapter{RootPath: targetDir}
	println("=== 文件目录树形结构 ===")
	TreeAdapter.PrintTree(dirTreeAdapter, dirTreeAdapter.GetRootNode(), "", true)
}

func _LogOn(ws *workspace.Workspace, parts []string) {
	targetEditor := getTargetEditor(ws, parts) // 解析目标文件（见下方辅助函数）
	if targetEditor == nil {
		fmt.Println("错误：文件未找到或无活动文件")
		return
	}
	targetEditor.SetLogEnabled(true)
	fmt.Printf("已为文件 %s 启用日志\n", targetEditor.GetFilePath())
}

// 处理log-off：关闭指定文件/当前活动文件的日志
func _LogOff(ws *workspace.Workspace, parts []string) {
	targetEditor := getTargetEditor(ws, parts)
	if targetEditor == nil {
		fmt.Println("错误：文件未找到或无活动文件")
		return
	}
	targetEditor.SetLogEnabled(false)
	fmt.Printf("已关闭文件 %s 的日志\n", targetEditor.GetFilePath())
}
func GetAfterLastBackslash(s string) string {
	idx := strings.LastIndex(s, "\\")
	if idx == -1 {
		return ""
	}
	return s[idx+1:]
}

// 处理log-show：显示指定文件/当前活动文件的日志
func _LogShow(ws *workspace.Workspace, parts []string) {
	targetEditor := getTargetEditor(ws, parts)
	if targetEditor == nil {
		fmt.Println("错误：文件未找到或无活动文件")
		return
	}

	//fmt.Printf("s%",logFilePath)

	// 打印原始文件路径和计算的日志路径（用于调试）
	fmt.Printf("调试：目标文件路径 = %q\n", targetEditor.GetFilePath())
	// logFilePath := "." + filePath + ".log"
	logFilePath := "." + GetAfterLastBackslash(targetEditor.GetFilePath()) + ".log"
	fmt.Printf("调试：日志文件路径 = %q\n", "logs\\"+logFilePath) // 检查路径是否正确

	content, err := os.ReadFile("logs\\" + logFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("日志文件不存在：%s\n", "logs\\"+logFilePath)
			return
		}
		fmt.Printf("读取日志失败：%v\n", err)
		return
	}
	fmt.Printf("===== 日志内容（%s） =====\n", "logs\\"+logFilePath)
	fmt.Print(string(content))
}

// 辅助函数：获取目标文件的编辑器（支持指定文件或当前活动文件）
func getTargetEditor(ws *workspace.Workspace, parts []string) common.Editor {
	if len(parts) >= 2 {
		// 指定文件：从已打开的编辑器中查找
		if _editor, exists := ws.OpenEditors["files\\"+parts[1]]; exists {
			return _editor
		}
		return nil
	} else {
		// 无参数：使用当前活动文件
		return ws.GetActiveEditor()
	}
}

func _Save(ws *workspace.Workspace, input string, debug bool, parts []string) {
	if debug {
		fmt.Printf("[DEBUG] 进入 save 命令处理，输入: %q，参数拆分: %v\n", input, parts)
	}

	// 1. 处理无参数：保存当前活动文件
	if len(parts) == 1 {
		if debug {
			fmt.Println("[DEBUG] 无参数，尝试保存当前活动文件")
		}
		activeEditor := ws.GetActiveEditor()
		if activeEditor == nil {
			if debug {
				fmt.Println("[DEBUG] 未找到活动文件")
			}
			fmt.Println("没有活动文件可保存")
			return
		}
		if debug {
			fmt.Printf("[DEBUG] 找到活动文件: %s，准备保存\n", activeEditor.GetFilePath())
		}
		if err := ws.SaveFile(activeEditor); err != nil {
			if debug {
				fmt.Printf("[DEBUG] 活动文件保存失败: %v\n", err)
			}
			fmt.Printf("保存失败: %v\n", err)
		} else {
			if debug {
				fmt.Printf("[DEBUG] 活动文件保存成功: %s\n", activeEditor.GetFilePath())
			}
			fmt.Printf("已保存活动文件: %s\n", activeEditor.GetFilePath())
		}
		return
	}

	// 2. 处理参数：保存指定文件或所有文件
	subCmd := parts[1]
	if debug {
		fmt.Printf("[DEBUG] 检测到子命令: %q\n", subCmd)
	}
	switch subCmd {
	case "all":
		if debug {
			fmt.Println("[DEBUG] 处理 save all，准备保存所有打开的文件")
		}
		// 保存所有已打开的文件
		openEditors := ws.GetOpenEditors()
		if len(openEditors) == 0 {
			if debug {
				fmt.Println("[DEBUG] 未找到任何打开的文件")
			}
			fmt.Println("没有打开的文件可保存")
			return
		}
		if debug {
			fmt.Printf("[DEBUG] 共找到 %d 个打开的文件，开始批量保存\n", len(openEditors))
		}
		successCount := 0
		for i, _editor := range openEditors {
			if debug {
				fmt.Printf("[DEBUG] 正在保存第 %d 个文件: %s\n", i+1, _editor.GetFilePath())
			}
			if err := ws.SaveFile(_editor); err != nil {
				if debug {
					fmt.Printf("[DEBUG] 第 %d 个文件保存失败: %v\n", i+1, err)
				}
				fmt.Printf("保存文件 %s 失败: %v\n", _editor.GetFilePath(), err)
			} else {
				successCount++
				if debug {
					fmt.Printf("[DEBUG] 第 %d 个文件保存成功: %s\n", i+1, _editor.GetFilePath())
				}
			}
		}
		if debug {
			fmt.Printf("[DEBUG] 批量保存完成，成功 %d 个，失败 %d 个\n", successCount, len(openEditors)-successCount)
		}
		fmt.Printf("批量保存完成，成功 %d 个，失败 %d 个\n", successCount, len(openEditors)-successCount)

	default:
		// 保存指定文件（subCmd 为文件路径）
		targetPath := subCmd
		if debug {
			fmt.Printf("[DEBUG] 处理指定文件保存，目标路径: %s\n", targetPath)
		}
		// 检查文件是否已打开
		openEditors := ws.GetOpenEditors()
		var targetEditor common.Editor
		for _, _editor := range openEditors {
			if _editor.GetFilePath() == targetPath {
				targetEditor = _editor
				if debug {
					fmt.Printf("[DEBUG] 在已打开文件中找到目标文件: %s\n", targetPath)
				}
				break
			}
		}
		if targetEditor == nil {
			if debug {
				fmt.Printf("[DEBUG] 目标文件 %s 未打开\n", targetPath)
			}
			fmt.Printf("文件 %s 未打开，无法保存\n", targetPath)
			return
		}
		// 执行保存
		if err := ws.SaveFile(targetEditor); err != nil {
			if debug {
				fmt.Printf("[DEBUG] 指定文件 %s 保存失败: %v\n", targetPath, err)
			}
			fmt.Printf("保存文件 %s 失败: %v\n", targetPath, err)
		} else {
			if debug {
				fmt.Printf("[DEBUG] 指定文件 %s 保存成功\n", targetPath)
			}
			fmt.Printf("已保存文件: %s\n", targetPath)
		}
	}
}

func _init(ws *workspace.Workspace, input string) {
	// 使用 strings.Fields 自动合并多个空格
	parts := strings.Fields(input)
	if len(parts) < 2 {
		fmt.Println("用法: init <filename> [with-log]")
		return
	}

	// 文件名包含扩展名
	fileName := parts[1]
	withLog := len(parts) >= 3 && parts[2] == "with-log"

	// 获取文件后缀，决定初始化内容
	ext := strings.ToLower(filepath.Ext(fileName))
	var content string
	switch ext {
	case ".txt":
		if withLog {
			content = "# log\n"
		}
	case ".xml":
		content = `<?xml version="1.0" encoding="UTF-8"?>
<root id="root">
</root>
`
	default:
		fmt.Printf("不支持的文件类型: %s（仅支持 .txt/.xml）\n", ext)
		return
	}

	// 生成完整文件路径（暂未保存）
	fullPath := "files\\" + fileName

	// 创建未保存的缓冲区
	_editor := editor.NewTextEditor(fullPath, content, ws)
	_editor.MarkAsModified(true)

	// 添加到工作区并设为活动文件
	ws.OpenEditors[_editor.GetFilePath()] = _editor
	ws.SetActiveEditor(_editor)

	fmt.Printf("已创建新缓冲区: %s（未保存）\n", fullPath)
	if withLog && ext == ".txt" {
		fmt.Println("已自动添加日志标记 '# log'")
	}
}

// generateDirectoryTree 生成指定目录的树形结构字符串
func generateDirectoryTree(rootDir string) (string, error) {
	// 获取目录下的所有条目（文件和子目录）
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	// 递归构建目录树
	buildTree(rootDir, entries, "", true, &builder)
	return builder.String(), nil
}

func buildTree(root string, entries []os.DirEntry, prefix string, isLast bool, builder *strings.Builder) {
	for i, entry := range entries {
		// 判断是否为最后一个条目
		isCurrentLast := i == len(entries)-1

		// 绘制前缀和连接线（修复根目录第一个条目格式）
		builder.WriteString(prefix)
		if isCurrentLast {
			builder.WriteString("└── ")
		} else {
			builder.WriteString("├── ")
		}

		// 写入条目名称
		builder.WriteString(entry.Name())
		builder.WriteString("\n")

		// 递归处理子目录（保持不变）
		if entry.IsDir() {
			var childPrefix string
			if prefix == "" {
				if isCurrentLast {
					childPrefix = "    "
				} else {
					childPrefix = "│   "
				}
			} else {
				if isCurrentLast {
					childPrefix = prefix + "    "
				} else {
					childPrefix = prefix + "│   "
				}
			}

			subDir := filepath.Join(root, entry.Name())
			subEntries, err := os.ReadDir(subDir)
			if err != nil {
				continue
			}
			buildTree(subDir, subEntries, childPrefix, isCurrentLast, builder)
		}
	}
}

func _EditorList(ws *workspace.Workspace) {
	openEditors := ws.GetOpenEditors()
	if len(openEditors) == 0 {
		fmt.Printf("error:")
	}

	//j := len(openEditors)
	//fmt.Printf("\n")
	//modified := false //是否修改
	for _, _editor := range openEditors {
		if _editor.GetFilePath() != "" {
			if _editor.IsModified() {
				fmt.Printf("%s [modified] (%s)\n", _editor.GetFilePath(),
					timeStatistics.GetFormattedDuration(_editor.GetFilePath()))
			} else {
				fmt.Printf("%s (%s)\n", _editor.GetFilePath(),
					timeStatistics.GetFormattedDuration(_editor.GetFilePath()))
			}
		}
	}

}

func _edit(ws *workspace.Workspace, parts []string) {
	if len(parts) < 2 {
		fmt.Printf("请指定文件:edit [file]\n")
		return
	}
	fileName := parts[1]
	if fileName == "" {
		fmt.Printf("请指定文件:edit [file]\n")
	} else {
		fileName = "files\\" + fileName
		_, exists := ws.OpenEditors[fileName]
		if exists {
			activeFileName := ws.GetActiveEditor().GetFilePath()
			if activeFileName == "" {
				fmt.Println("error")
				return
			}
			ws.NotifyObservers(common.WorkspaceEvent{
				FilePath:  activeFileName,
				Type:      common.EventFileSwitched,
				Command:   "",
				Timestamp: time.Now().UnixMilli(),
			})
			ws.SetActiveEditor(ws.OpenEditors[fileName])
		} else {
			fmt.Printf("文件未打开: [file]\n")
		}
	}

}

func _show(ws *workspace.Workspace, parts []string) {
	activeEditor, ok := ws.GetActiveEditor().(*editor.TextEditor)
	if !ok {
		fmt.Println("断言失败")
	}
	if activeEditor == nil {
		fmt.Println("没有活动文件")
		return
	}
	if len(parts) == 1 {
		fmt.Printf("指令格式错误:show [startLine:endLine]\n")
		return
	}
	startLine, endLine := 0, 0
	if len(parts) > 0 {
		rangeStr := parts[1]
		// 按 ":" 分割字符串，处理 "start:end" 格式
		segments := strings.Split(rangeStr, ":")
		if len(segments) != 2 {
			fmt.Println("参数格式错误，应为 show [startLine:endLine]")
			return
		}

		// 解析起始行（必须为正整数）
		s, err := strconv.Atoi(segments[0])
		if err != nil || s < 1 {
			fmt.Println("起始行必须为正整数")
			return
		}

		// 解析结束行（必须为正整数且不小于起始行）
		e, err := strconv.Atoi(segments[1])
		if err != nil || e < 1 {
			fmt.Println("结束行必须为正整数")
			return
		}
		if e < s {
			fmt.Println("结束行不能小于起始行")
			return
		}

		startLine, endLine = s, e
		// 调用编辑器的 Show 方法
		activeEditor.Show(startLine, endLine)
	}

	// 调用编辑器的 Show 方法
	//activeEditor.Show(startLine, endLine)
}

func _append(ws *workspace.Workspace, parts []string) {
	// 1. 校验活动文件是否存在
	activeEditor, ok := ws.GetActiveEditor().(*editor.TextEditor)
	if !ok {
		fmt.Println("editor 断言失败")
	}
	if activeEditor == nil {
		fmt.Println("错误：没有打开的文件，请先使用 load 命令加载文件")
		return
	}

	// 2. 解析参数：实际参数是 parts[1:]（排除 parts[0] 的 "append"）
	// 检查是否提供了参数（至少需要一个参数片段）
	if len(parts) < 2 { // parts 长度至少为 2（["append", "参数"]）
		fmt.Println("参数错误：请指定要追加的文本，格式为 append \"text\"")
		return
	}

	// 提取实际参数部分（排除命令名），并合并成完整字符串
	// 例如 parts[1:] 是 ["\"hello", "world\""] → 合并为 "\"hello world\""
	textArg := strings.Join(parts[1:], " ")

	// 3. 校验文本是否用双引号包裹
	if len(textArg) < 2 || textArg[0] != '"' || textArg[len(textArg)-1] != '"' {
		fmt.Println("参数错误：文本必须用双引号包裹，格式为 append \"text\"")
		return
	}

	// 4. 提取引号内的文本（去除首尾引号）
	content := textArg[1 : len(textArg)-1]

	// 5. 执行追加操作
	activeEditor.Append(content)
	fmt.Printf("已在文件末尾追加一行：%s\n", content)

}

func _insert(ws *workspace.Workspace, parts []string) {
	// 1. 校验活动文件是否存在
	activeEditor, ok := ws.GetActiveEditor().(*editor.TextEditor)
	if !ok {
		fmt.Println("断言失败")
	}
	if activeEditor == nil {
		fmt.Println("错误：没有打开的文件，请先使用 load 命令加载文件")
		return
	}

	// 2. 校验参数数量    // 格式要求：至少需要两个参数（位置 <line:col> 和带引号的文本）
	if len(parts) < 3 {
		fmt.Println("参数错误：格式为 insert <line:col> \"text\"（例如 insert 1:4 \"XYZ\"）")
		return
	}

	// 3. 解析位置参数 <line:col>
	posStr := parts[1]
	var line, col int
	// 按 ":" 分割行号和列号
	posParts := strings.Split(posStr, ":")
	if len(posParts) != 2 {
		fmt.Println("参数错误：位置格式应为 line:col（例如 1:4）")
		return
	}
	// 转换行号为整数（1-based）
	line, err := strconv.Atoi(posParts[0])
	if err != nil || line < 1 {
		fmt.Println("参数错误：行号必须为正整数")
		return
	}
	// 转换列号为整数（1-based）
	col, err = strconv.Atoi(posParts[1])
	if err != nil || col < 1 {
		fmt.Println("参数错误：列号必须为正整数")
		return
	}

	// 4. 解析插入文本（合并后续参数，支持带空格和换行符）
	// 文本参数从 parts[2] 开始（排除命令名和位置参数）
	textParts := parts[2:]
	textArg := strings.Join(textParts, " ")
	// 校验文本是否用双引号包裹
	if len(textArg) < 2 || textArg[0] != '"' || textArg[len(textArg)-1] != '"' {
		fmt.Println("参数错误：文本必须用双引号包裹（例如 \"XYZ\"）")
		return
	}
	// 提取引号内的文本（支持包含换行符 \n）
	content := textArg[1 : len(textArg)-1]

	// 5. 执行插入操作（调用编辑器的 Insert 方法）
	activeEditor.Insert(line, col, content)
	fmt.Printf("已在 %d:%d 位置插入文本：%s\n", line, col, content)
}

func _delete(ws *workspace.Workspace, parts []string) {
	// 1. 校验活动文件是否存在
	activeEditor, ok := ws.GetActiveEditor().(*editor.TextEditor)
	if !ok {
		fmt.Println("断言失败")
	}
	if activeEditor == nil {
		fmt.Println("错误：没有打开的文件，请先使用 load 命令加载文件")
		return
	}

	// 2. 校验参数数量（必须包含 <line:col> 和 <len> 两个参数）
	if len(parts) != 3 {
		fmt.Println("参数错误：格式为 delete <line:col> <len>（例如 delete 1:7 5）")
		return
	}

	// 3. 解析位置参数 <line:col>
	posStr := parts[1]
	var line, col int
	posParts := strings.Split(posStr, ":")
	if len(posParts) != 2 {
		fmt.Println("参数错误：位置格式应为 line:col（例如 1:7）")
		return
	}
	// 行号必须为正整数
	line, err := strconv.Atoi(posParts[0])
	if err != nil || line < 1 {
		fmt.Println("参数错误：行号必须为正整数")
		return
	}
	// 列号必须为正整数
	col, err = strconv.Atoi(posParts[1])
	if err != nil || col < 1 {
		fmt.Println("参数错误：列号必须为正整数")
		return
	}

	// 4. 解析删除长度 <len>
	lenStr := parts[2]
	length, err := strconv.Atoi(lenStr)
	if err != nil || length < 1 {
		fmt.Println("参数错误：删除长度必须为正整数")
		return
	}

	// 5. 执行删除操作（调用编辑器的 Delete 方法）
	// 编辑器内部会处理：行号/列号越界、删除长度超出行尾等异常
	activeEditor.Delete(line, col, length)
	fmt.Printf("已从 %d:%d 位置删除 %d 个字符\n", line, col, length)
}

func _replace(ws *workspace.Workspace, parts []string) {
	// 1. 校验活动文件是否存在
	activeEditor, ok := ws.GetActiveEditor().(*editor.TextEditor)
	if !ok {
		fmt.Println("断言失败")
	}
	if activeEditor == nil {
		fmt.Println("错误：没有打开的文件，请先使用 load 命令加载文件")
		return
	}

	// 2. 校验参数数量（必须包含 <line:col>、<len>、"text" 三个参数）
	if len(parts) < 4 {
		fmt.Println("参数错误：格式为 replace <line:col> <len> \"text\"（例如 replace 1:1 4 \"slow\"）")
		return
	}

	// 3. 解析位置参数 <line:col>
	posStr := parts[1]
	var line, col int
	//posParts按 ":" 分割行号和列号
	posParts := strings.Split(posStr, ":")
	if len(posParts) != 2 {
		fmt.Println("参数错误：位置格式应为 line:col（例如 1:1）")
		return
	}
	// 行号必须为正整数
	line, err := strconv.Atoi(posParts[0])
	if err != nil || line < 1 {
		fmt.Println("参数错误：行号必须为正整数")
		return
	}
	// 列号必须为正整数
	col, err = strconv.Atoi(posParts[1])
	if err != nil || col < 1 {
		fmt.Println("参数错误：列号必须为正整数")
		return
	}

	// 4. 解析删除删除长度 <len>
	lenStr := parts[2]
	length, err := strconv.Atoi(lenStr)
	if err != nil || length < 1 {
		fmt.Println("参数错误：删除长度必须为正整数")
		return
	}

	// 5. 解析替换文本（支持带空格和空字符串）
	// 文本参数从 parts[3] 开始，合并所有后续片段
	textParts := parts[3:]
	textArg := strings.Join(textParts, " ")
	// 校验文本是否用双引号包裹（空字符串需表示为 ""）
	if len(textArg) < 2 || textArg[0] != '"' || textArg[len(textArg)-1] != '"' {
		fmt.Println("参数错误：替换文本必须用双引号包裹（例如 \"slow\" 或 \"\"）")
		return
	}
	// 提取引号内的文本（支持空字符串）
	content := textArg[1 : len(textArg)-1]

	// 6. 执行替换操作（调用编辑器的 Replace 方法）
	// 编辑器内部会先执行 delete 再执行 insert，处理各类异常
	activeEditor.Replace(line, col, length, content)
	fmt.Printf("已从 %d:%d 位置替换 %d 个字符为：%s\n", line, col, length, content)
}

func _insertBefore(ws *workspace.Workspace, parts []string) {
	// 1. 参数校验：至少需要4个参数（指令名+tag+newId+targetId），可选text
	if len(parts) < 4 {
		fmt.Println("参数错误：insert-before 指令格式为 insert-before <tag> <newId> <targetId> [text]")
		return
	}
	tag := parts[1]
	newId := parts[2]
	targetId := parts[3]
	text := ""
	if len(parts) >= 5 {
		text = strings.Join(parts[4:], " ") // 处理带空格的text参数
	}
	text = strings.TrimSpace(text)
	if text != "" {
		text = text[1 : len(text)-1]
	}
	// 2. 获取当前编辑器并校验类型
	_editor := ws.GetActiveEditor()
	if _editor == nil {
		fmt.Println("错误：未打开任何文件")
		return
	}
	xmlEditor, ok := _editor.(*editor.XmlEditor)
	if !ok {
		fmt.Println("错误：当前打开的不是XML文件，无法执行insert-before操作")
		return
	}

	// 3. 执行操作
	err := xmlEditor.InsertBefore(tag, newId, targetId, text)
	if err != nil {
		fmt.Println(err)
	}
}

// _appendChild 处理 append-child <tag> <newId> <parentId> ["text"] 指令
func _appendChild(ws *workspace.Workspace, parts []string) {
	if len(parts) < 4 {
		fmt.Println("参数错误：append-child 指令格式为 append-child <tag> <newId> <parentId> [text]")
		return
	}
	//fmt.Printf("parts1:%s", parts[1])
	tag := parts[1]
	newId := parts[2]
	parentId := parts[3]
	text := ""
	if len(parts) >= 5 {
		text = strings.Join(parts[4:], " ")
	}
	//fmt.Println(tag, newId, parentId, text)
	//fmt.Printf("1233")
	//fmt.Println(text)
	text = strings.TrimSpace(text)
	if text != "" {
		text = text[1 : len(text)-1]
	}
	_editor := ws.GetActiveEditor()
	if _editor == nil {
		fmt.Println("错误：未打开任何文件")
		return
	}
	xmlEditor, ok := _editor.(*editor.XmlEditor)
	if !ok {
		fmt.Println("错误：当前打开的不是XML文件，无法执行append-child操作")
		return
	}

	err := xmlEditor.AppendChild(tag, newId, parentId, text)
	if err != nil {
		fmt.Println(err)
	}
}

// _editId 处理 edit-id <oldId> <newId> 指令
func _editId(ws *workspace.Workspace, parts []string) {
	if len(parts) != 3 {
		fmt.Println("参数错误：edit-id 指令格式为 edit-id <oldId> <newId>")
		return
	}
	oldId := parts[1]
	newId := parts[2]

	_editor := ws.GetActiveEditor()
	if _editor == nil {
		fmt.Println("错误：未打开任何文件")
		return
	}
	xmlEditor, ok := _editor.(*editor.XmlEditor)
	if !ok {
		fmt.Println("错误：当前打开的不是XML文件，无法执行edit-id操作")
		return
	}

	err := xmlEditor.EditId(oldId, newId)
	if err != nil {
		fmt.Println(err)
	}
}

// _editText 处理 edit-text <elementId> ["text"] 指令
func _editText(ws *workspace.Workspace, parts []string) {

	if len(parts) < 2 {
		fmt.Println("参数错误：edit-text 指令格式为 edit-text <elementId> [text]")
		return
	}
	elementId := parts[1]
	text := ""
	if len(parts) >= 3 {
		text = strings.Join(parts[2:], " ")
	}
	text = strings.TrimSpace(text)
	//fmt.Println(parts[1])
	//fmt.Println(text)
	_editor := ws.GetActiveEditor()
	if _editor == nil {
		fmt.Println("错误：未打开任何文件")
		return
	}
	xmlEditor, ok := _editor.(*editor.XmlEditor)
	if !ok {
		fmt.Println("错误：当前打开的不是XML文件，无法执行edit-text操作")
		return
	}

	err := xmlEditor.EditText(elementId, text)
	if err != nil {
		fmt.Println(err)
	}
}

// _xmlDelete 处理 delete <elementId> 指令（XML版）
// 返回值：true表示已按XML指令处理，false表示不是XML编辑器，需走文本delete逻辑
func _xmlDelete(ws *workspace.Workspace, parts []string) bool {
	if len(parts) != 2 {
		return false // 参数个数不对，交给文本delete处理
	}
	elementId := parts[1]

	_editor := ws.GetActiveEditor()
	if _editor == nil {
		fmt.Println("错误：未打开任何文件")
		return true
	}
	xmlEditor, ok := _editor.(*editor.XmlEditor)
	if !ok {
		return false // 不是XML编辑器，交给文本delete处理
	}

	err := xmlEditor.Delete(elementId)
	if err != nil {
		fmt.Println(err)
	}
	return true
}

// _xmlTree 处理 xml-tree [file] 指令
func _xmlTree(ws *workspace.Workspace, parts []string) {

	if len(parts) < 2 {
		targetEditor := ws.GetActiveEditor()
		if targetEditor == nil {
			fmt.Println("当前没有活跃的编辑器")
			return
		}
		// 类型断言：失败则直接返回，避免后续 nil 调用
		xmlEditor, ok := targetEditor.(*editor.XmlEditor)
		if !ok {
			fmt.Printf("活跃编辑器类型错误，非 XML 编辑器\n")
			return
		}

		// 跨平台路径拼接：使用 filepath.Join 替代硬编码的 \
		err := xmlEditor.XmlTree(targetEditor.GetFilePath())
		if err != nil {
			fmt.Printf("生成 XML 树失败：%v\n", err)
			return
		}
		return
	}

	// 分支2：有额外参数，拼接目标文件路径
	filePath := strings.TrimSpace(strings.Join(parts[1:], ""))
	if filePath == "" {
		fmt.Println("文件路径参数为空")
		return
	}
	totalFilePath := filepath.Join("files", filePath)

	// 遍历已打开的编辑器，查找目标文件
	var targetXmlEditor *editor.XmlEditor
	for _, ed := range ws.OpenEditors {
		if ed.GetFilePath() == totalFilePath {
			// 类型断言：失败则提示并继续遍历（可能有其他编辑器）
			xmlEd, ok := ed.(*editor.XmlEditor)
			if !ok {
				fmt.Printf("编辑器文件 %s 非 XML 编辑器类型\n", totalFilePath)
				continue
			}
			targetXmlEditor = xmlEd
			break
		}
	}

	// 处理未找到编辑器/文件的情况
	if targetXmlEditor == nil {
		fmt.Printf("未找到已打开的 XML 编辑器：%s\n", totalFilePath)
		return
	}

	// 生成并打印 XML 树（此处假设 XmlEditor 也有 XmlTree 方法，若逻辑不同需调整）
	err := targetXmlEditor.XmlTree(totalFilePath)
	if err != nil {
		fmt.Printf("生成 XML 树失败：%v\n", err)
		return
	}

}

func _xmlTreeV2(ws *workspace.Workspace, parts []string) {
	// 1. 获取目标 XML 文件路径
	var filePath string
	if len(parts) < 2 {
		activeEditor := ws.GetActiveEditor()
		if activeEditor == nil {
			fmt.Println("当前没有活跃的编辑器")
			return
		}
		filePath = activeEditor.GetFilePath()
	} else {
		// 假设外部参数是相对于 "files" 目录的路径
		filePath = filepath.Join("files", strings.TrimSpace(strings.Join(parts[1:], "")))
	}

	// 2. 读取并解析 XML 文件
	xmlFile, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("无法打开 XML 文件: %v\n", err)
		return
	}

	var rootXML TreeAdapter.XMLNode
	err = xml.Unmarshal(xmlFile, &rootXML)
	if err != nil {
		fmt.Printf("解析 XML 失败: %v\n", err)
		return
	}

	// 3. 使用适配器
	xmlAdapter := &TreeAdapter.XMLTreeAdapter{RootXML: rootXML}

	fmt.Printf("=== XML 树形结构 [%s] ===\n", filePath)

	// 4. 调用通用的打印函数
	// 注意：初始调用 prefix 为 ""，isLast 为 true（因为根节点只有一个）
	TreeAdapter.PrintTree(xmlAdapter, xmlAdapter.GetRootNode(), "", true)
}
