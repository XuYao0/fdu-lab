package editor

import (
	"fmt"
	"lab1/common"
	"strconv"
	"strings"
	"time"
)

// 暴露给外部的操作方法（供用户指令调用）

func (te *TextEditor) Append(text string) {
	if te.logEnabled {
		te.workspaceApi.NotifyObservers(common.WorkspaceEvent{
			FilePath:  te.GetFilePath(),
			Type:      "Append",
			Command:   "Append " + text,
			Timestamp: time.Now().UnixMilli(),
		})
	}

	te.ExecuteCommand(NewAppendCommand(te, text))
}

func (te *TextEditor) Insert(line, col int, text string) {
	if te.logEnabled {
		commandStr := "Insert " + strconv.Itoa(line) + "," + strconv.Itoa(col) + " " + text
		te.workspaceApi.NotifyObservers(common.WorkspaceEvent{
			FilePath:  te.GetFilePath(),
			Type:      "Insert",
			Command:   commandStr,
			Timestamp: time.Now().UnixMilli(),
		})
	}
	te.ExecuteCommand(NewInsertCommand(te, line, col, text))
}

func (te *TextEditor) Delete(line, col, length int) {
	if te.logEnabled {
		commandStr := "Delete " + strconv.Itoa(line) + "," + strconv.Itoa(col) + "," + strconv.Itoa(length)
		te.workspaceApi.NotifyObservers(common.WorkspaceEvent{
			FilePath:  te.GetFilePath(),
			Type:      "Delete",
			Command:   commandStr,
			Timestamp: time.Now().UnixMilli(),
		})
	}
	te.ExecuteCommand(NewDeleteCommand(te, line, col, length))
}

func (te *TextEditor) Replace(line, col, length int, text string) {
	if te.logEnabled {
		commandStr := "Replace " + strconv.Itoa(line) + "," + strconv.Itoa(col) + "," + strconv.Itoa(length) + " " + text
		te.workspaceApi.NotifyObservers(common.WorkspaceEvent{
			FilePath:  te.GetFilePath(),
			Type:      "Replace",
			Command:   commandStr,
			Timestamp: time.Now().UnixMilli(),
		})
	}
	te.ExecuteCommand(NewReplaceCommand(te, line, col, length, text))
}

func (te *TextEditor) Show(startLine, endLine int) {
	if te.logEnabled {
		commandStr := "Show " + strconv.Itoa(startLine) + "," + strconv.Itoa(endLine)
		te.workspaceApi.NotifyObservers(common.WorkspaceEvent{
			FilePath:  te.GetFilePath(),
			Type:      "Show",
			Command:   commandStr,
			Timestamp: time.Now().UnixMilli(),
		})
	}

	lineCount := len(te.lines)

	// 处理空文件
	if lineCount == 0 {
		fmt.Println("(空文件)")
		return
	}

	// 解析行范围（默认显示全文）
	actualStart := 1
	actualEnd := lineCount

	if startLine > 0 {
		// 修正起始行越界（最小为1）
		actualStart = startLine
		//if actualStart < 1 {
		//	actualStart = 1
		//}
		// 修正起始行超过总行数（视为无效范围）
		if actualStart > lineCount {
			fmt.Println("起始行超出文件范围")
			return
		}

		// 修正结束行（默认到最后一行，最大为总行数）
		if endLine > 0 {
			actualEnd = endLine
			if actualEnd > lineCount {
				actualEnd = lineCount
			}
			// 起始行不能大于结束行
			if actualStart > actualEnd {
				fmt.Println("起始行不能大于结束行")
				return
			}
		}
	}

	// 计算行号宽度（用于对齐）
	maxLineNum := actualEnd
	if lineCount > maxLineNum {
		maxLineNum = lineCount // 确保行号宽度适配最大行号
	}
	lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))
	lineFormat := fmt.Sprintf("%%%dd: %%s\n", lineNumWidth) // 格式：  1: Hello

	//
	// 拼接输出内容
	var output strings.Builder
	for i := actualStart - 1; i < actualEnd; i++ { // 转换为 0-based 索引
		lineNum := i + 1
		output.WriteString(fmt.Sprintf(lineFormat, lineNum, te.lines[i]))
	}

	// 打印结果（去除末尾多余换行）
	fmt.Print(output.String())
}

//XML

func (x *XmlEditor) InsertBefore(tag, newId, targetId, text string) error {
	// 日志通知（兼容Lab1的观察者）
	if x.logEnabled {
		commandStr := fmt.Sprintf("insert-before %s %s %s %s", tag, newId, targetId, text)
		if x.logEnabled {
			x.workspaceApi.NotifyObservers(common.WorkspaceEvent{
				FilePath:  x.GetFilePath(),
				Type:      "InsertBefore",
				Command:   commandStr,
				Timestamp: time.Now().UnixMilli(),
			})
		}
	}
	// 执行命令
	//cmd := NewInsertBeforeCommand(x, tag, newId, targetId, text)
	x.ExecuteCommand(NewInsertBeforeCommand(x, tag, newId, targetId, text))
	// 处理执行错误
	//if err := cmd.Execute(); err != nil {
	//	fmt.Printf("插入元素失败: %v\n", err)
	//}
	return nil
}

func (xe *XmlEditor) AppendChild(tag, newId, parentId, text string) error {
	if xe.logEnabled {
		commandStr := fmt.Sprintf("append-child %s %s %s %s", tag, newId, parentId, text)
		if xe.logEnabled {
			xe.workspaceApi.NotifyObservers(common.WorkspaceEvent{
				FilePath:  xe.GetFilePath(),
				Type:      "AppendChild",
				Command:   commandStr,
				Timestamp: time.Now().UnixMilli(),
			})
		}
	}
	cmd := NewAppendChildCommand(xe, tag, newId, parentId, text)
	fmt.Println(cmd)
	xe.ExecuteCommand(NewAppendChildCommand(xe, tag, newId, parentId, text))
	fmt.Println(xe.lines)
	//if err := cmd.Execute(); err != nil {
	//	//fmt.Printf("追加子元素失败: %v\n", err)
	//	return err
	//}
	return nil
}

func (x *XmlEditor) EditId(oldId, newId string) error {
	if x.logEnabled {
		commandStr := fmt.Sprintf("edit-id %s %s", oldId, newId)
		if x.logEnabled {
			x.workspaceApi.NotifyObservers(common.WorkspaceEvent{
				FilePath:  x.GetFilePath(),
				Type:      "EditId",
				Command:   commandStr,
				Timestamp: time.Now().UnixMilli(),
			})
		}
	}
	//cmd := NewEditIdCommand(x, oldId, newId)
	x.ExecuteCommand(NewEditIdCommand(x, oldId, newId))
	//if err := cmd.Execute(); err != nil {
	//	fmt.Printf("修改元素ID失败: %v\n", err)
	//}
	return nil
}

func (x *XmlEditor) EditText(elementId, text string) error {
	if x.logEnabled {
		commandStr := fmt.Sprintf("edit-text %s %s", elementId, text)
		if x.logEnabled {
			x.workspaceApi.NotifyObservers(common.WorkspaceEvent{
				FilePath:  x.GetFilePath(),
				Type:      "EditText",
				Command:   commandStr,
				Timestamp: time.Now().UnixMilli(),
			})
		}
	}
	//cmd := NewEditTextCommand(x, elementId, text)
	x.ExecuteCommand(NewEditTextCommand(x, elementId, text))
	//if err := cmd.Execute(); err != nil {
	//	fmt.Printf("修改元素文本失败: %v\n", err)
	//}
	return nil
}

func (x *XmlEditor) Delete(elementId string) error {
	if x.logEnabled {
		commandStr := fmt.Sprintf("delete %s", elementId)
		if x.logEnabled {
			x.workspaceApi.NotifyObservers(common.WorkspaceEvent{
				FilePath:  x.GetFilePath(),
				Type:      "Delete",
				Command:   commandStr,
				Timestamp: time.Now().UnixMilli(),
			})
		}
	}
	//cmd := NewDeleteCommand(xe, elementId)
	x.ExecuteCommand(NewXmlDeleteCommand(x, elementId))
	return nil
}

func (x *XmlEditor) XmlTree(filePath string) error {
	// 日志通知
	if x.logEnabled {
		commandStr := "xml-tree"
		if filePath != "" {
			commandStr += " " + filePath
		}
		//fmt.Println(filePath)
		if x.logEnabled {
			x.workspaceApi.NotifyObservers(common.WorkspaceEvent{
				FilePath:  x.GetFilePath(),
				Type:      "XmlTree",
				Command:   commandStr,
				Timestamp: time.Now().UnixMilli(),
			})
		}
	}
	//默认就是显示当前，如果要显示其他的，在外层处理自己切换编辑器
	var targetEditor *XmlEditor
	if filePath != "" && filePath != x.GetFilePath() {
		fmt.Printf("暂未实现加载外部文件%s的逻辑，显示当前文件树形结构\n", filePath)
		targetEditor = x
	} else {
		targetEditor = x
	}

	// 显示树形结构（复用之前的GetContent方法，已实现树形输出）
	fmt.Println(targetEditor.GetTreeContent())
	return nil
}
