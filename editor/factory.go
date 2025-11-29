package editor

import (
	"errors"
	"fmt"
	"lab1/common"
	"os"
	"path/filepath"
	"strings"
)

// EditorFactory 编辑器工厂函数（适配 workspace.LoadFile/RestoreState 的 editorFactory 参数）
// 参数：path（文件路径）
// 返回：workspace.Editor 接口实例（文本编辑器）、错误信息
//
//	func EditorFactory(path string) (workspace.Editor, error) {
//		// 1. 校验文件是否存在
//		if _, err := os.Stat(path); err != nil {
//			return nil, errors.New("file not found: " + path)
//		}
//
//		// 2. 根据文件后缀判断编辑器类型（当前仅支持 .txt 文本文件）
//		ext := strings.ToLower(filepath.Ext(path))
//		switch ext {
//		case ".txt":
//			// 3. 读取文件内容，创建文本编辑器
//			content, err := os.ReadFile(path)
//			if err != nil {
//				return nil, errors.New("read file failed: " + err.Error())
//			}
//			return NewTextEditor(path, string(content)), nil
//		default:
//			return nil, errors.New("unsupported file type: " + ext)
//		}
//	}
// func EditorFactory(path string, w *workspace.Workspace) (workspace.Editor, error) {

// 	isNewFile := false

// 	if _, err := os.Stat(path); err != nil {
// 		if os.IsNotExist(err) {
// 			// 新建空文件（权限 0644：所有者可读写，其他用户可读）
// 			if err := os.WriteFile(path, []byte(""), 0644); err != nil {
// 				return nil, fmt.Errorf("failed to create file: %w", err)
// 			}
// 			isNewFile = true // 标记为新创建的文件
// 			//w.AddInitUnsavedEditor()
// 		} else {
// 			// 如权限问题
// 			return nil, fmt.Errorf("failed to check file status: %w", err)
// 		}
// 	}

// 	content, err := os.ReadFile(path)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read file: %w", err)
// 	}

// 	ext := strings.ToLower(filepath.Ext(path))
// 	switch ext {
// 	case ".txt":
// 		editor := NewTextEditor(path, string(content))
// 		// 若为新创建的文件，标记为已修改
// 		if isNewFile {
// 			editor.MarkAsModified(true)
// 		}
// 		return editor, nil
// 	default:
// 		return nil, errors.New("unsupported file type: " + ext)
// 	}
// }

func EditorFactory(path string, wsApi common.WorkSpaceApi) (common.Editor, error) {
	isNewFile := false

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			// 新建空文件（权限 0644：所有者可读写，其他用户可读）
			if err := os.WriteFile(path, []byte(""), 0644); err != nil {
				return nil, fmt.Errorf("failed to create file: %w", err)
			}
			isNewFile = true // 标记为新创建的文件
		} else {

			return nil, fmt.Errorf("failed to check file status: %w", err)
		}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt":
		editor := NewTextEditor(path, string(content), wsApi)
		// 若为新创建的文件，标记为已修改且日志默认关闭
		if isNewFile {
			editor.MarkAsModified(true)
			editor.SetLogEnabled(false)
		} else {
			// 现有文件检查首行是否有# log标记
			firstLine := ""
			lines := strings.Split(string(content), "\n")
			if len(lines) > 0 {
				firstLine = strings.TrimSpace(lines[0])
			}

			logEnabled := strings.Contains(firstLine, "# log")
			editor.SetLogEnabled(logEnabled)
		}
		return editor, nil
	case ".xml":
		editor := NewXmlEditor(path, string(content), wsApi)
		if isNewFile {
			editor.MarkAsModified(true)
			editor.SetLogEnabled(false)
		} else {
			firstLine := ""
			lines := strings.Split(string(content), "\n")
			if len(lines) > 0 {
				firstLine = strings.TrimSpace(lines[0])
			}
			logEnabled := strings.Contains(firstLine, "# log")
			editor.SetLogEnabled(logEnabled)
		}
		return editor, nil
	default:
		return nil, errors.New("unsupported file type: " + ext)
	}
}

/*
新增逻辑，打开的时候，先检查首行# log标志，如果是新建的，那么默认lon为false
*/

//是维护两个状态集合，还是直接维护一个editor集合，用状态标识
// init后新建一个未保存类型的编辑器，和文件，直接改成活跃文件，
//然后可以load里调用init （没必要
