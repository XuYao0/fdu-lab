package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"lab1/common"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ------------------------------
// 观察者模式相关定义
// ------------------------------

// Observer 观察者接口（所有订阅工作区事件的模块需实现）

// ------------------------------
// 备忘录模式相关定义
// ------------------------------

// WorkspaceMemento 工作区状态备忘录（用于持久化）
type WorkspaceMemento struct {
	OpenedFilePaths   []string // 已打开文件路径列表
	ActiveFilePath    string   // 当前活动文件路径
	ModifiedFilePaths []string // 已修改文件路径列表
	FileStates        []FileState
}

//这里的文件日志状态切片，是需要修改的，因为真实的各种状态会动态变化，这里要加一个方法供调用

type FileState struct {
	FilePath   string
	LogEnabled bool // 该文件的日志开关状态
}

// ------------------------------
// 编辑器接口（工作区依赖此接口与编辑器交互）
// ------------------------------

// Workspace 工作区模块
type Workspace struct {
	OpenEditors map[string]common.Editor
	//UnsavedEditors map[string]Editor
	activeEditor common.Editor
	//isLogEnabled bool
	observers   []common.Observer
	mementoPath string
}

// NewWorkspace 创建工作区实例
func NewWorkspace(mementoPath string) *Workspace {
	return &Workspace{
		OpenEditors: make(map[string]common.Editor),
		//UnsavedEditors: make(map[string]Editor), // 初始化未保存缓冲区
		mementoPath: mementoPath,
	}
}

// ------------------------------
// 观察者模式实现
// ------------------------------

// RegisterObserver 注册观察者
func (w *Workspace) RegisterObserver(observer common.Observer) {
	w.observers = append(w.observers, observer)
}

// RemoveObserver 移除观察者
func (w *Workspace) RemoveObserver(observer common.Observer) {
	for i, obs := range w.observers {
		if obs == observer {
			// 从切片中移除
			w.observers = append(w.observers[:i], w.observers[i+1:]...)
			break
		}
	}
}

// NotifyObservers  通知所有观察者（公开，暴露给编辑器
func (w *Workspace) NotifyObservers(event common.WorkspaceEvent) {
	for _, observer := range w.observers {
		observer.Update(event)
	}
}

// ------------------------------
// 备忘录模式实现（状态持久化与恢复）
// ------------------------------

// CreateMemento 创建工作区状态备忘录
// func (w *Workspace) CreateMemento() *WorkspaceMemento {
// 	// 收集已打开文件路径
// 	openedPaths := make([]string, 0, len(w.OpenEditors))
// 	for path := range w.OpenEditors {
// 		openedPaths = append(openedPaths, path)
// 	}

// 	// 收集已修改文件路径
// 	modifiedPaths := make([]string, 0)
// 	for path, editor := range w.OpenEditors {
// 		if editor.IsModified() {
// 			modifiedPaths = append(modifiedPaths, path)
// 		}
// 	}

// 	// 活动文件路径
// 	activePath := ""
// 	if w.activeEditor != nil {
// 		activePath = w.activeEditor.GetFilePath()
// 	}

// 	return &WorkspaceMemento{
// 		OpenedFilePaths:   openedPaths,
// 		ActiveFilePath:    activePath,
// 		ModifiedFilePaths: modifiedPaths,
// 		//IsLogEnabled:      w.isLogEnabled,
// 		FileStates: ,
// 	}
// }

func (w *Workspace) CreateMemento() *WorkspaceMemento {
	// 收集已打开文件路径
	openedPaths := make([]string, 0, len(w.OpenEditors))
	for path := range w.OpenEditors {
		openedPaths = append(openedPaths, path)
	}

	// 收集已修改文件路径
	modifiedPaths := make([]string, 0)
	for path, editor := range w.OpenEditors {
		if editor.IsModified() {
			modifiedPaths = append(modifiedPaths, path)
		}
	}

	// 新增：收集每个文件的日志状态
	fileStates := make([]FileState, 0, len(w.OpenEditors))
	for path, editor := range w.OpenEditors {
		fileStates = append(fileStates, FileState{
			FilePath:   path,
			LogEnabled: editor.IsLogEnabled(), // 获取每个文件的日志开关状态
		})
	}

	// 活动文件路径
	activePath := ""
	if w.activeEditor != nil {
		activePath = w.activeEditor.GetFilePath()
	}

	return &WorkspaceMemento{
		OpenedFilePaths:   openedPaths,
		ActiveFilePath:    activePath,
		ModifiedFilePaths: modifiedPaths,
		FileStates:        fileStates, // 保存文件日志状态
	}
}

// SaveState 保存工作区状态到本地（持久化）
func (w *Workspace) SaveState() error {
	memento := w.CreateMemento()
	data, err := json.MarshalIndent(memento, "", "  ")
	if err != nil {
		return err
	}
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(w.mementoPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(w.mementoPath, data, 0644)
}

// RestoreState 从本地恢复工作区状态
func (w *Workspace) RestoreState(editorFactory func(path string, ws common.WorkSpaceApi) (common.Editor, error)) error {
	// 读取备忘录文件
	data, err := os.ReadFile(w.mementoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return err // 无状态文件，无需恢复
		}
		//return err
	}

	var memento WorkspaceMemento
	if err := json.Unmarshal(data, &memento); err != nil {
		return err
	}

	// 恢复日志开关
	//w.isLogEnabled = memento.IsLogEnabled

	// 恢复已打开文件（通过编辑器工厂创建对应类型的编辑器）
	for _, path := range memento.OpenedFilePaths {
		editor, err := editorFactory(path, w)
		if err != nil {
			return err
		}
		//w.NotifyObservers(common.WorkspaceEvent{
		//	FilePath:  path,
		//	Type:      common.EventFileActivated,
		//	Command:   "",
		//	Timestamp: time.Now().UnixMilli(),
		//})
		w.OpenEditors[path] = editor
	}

	// 恢复修改状态
	for _, path := range memento.ModifiedFilePaths {
		if editor, ok := w.OpenEditors[path]; ok {
			editor.MarkAsModified(true)
		}
	}
	//恢复日志状态
	logStateMap := make(map[string]bool)
	for _, state := range memento.FileStates {
		logStateMap[state.FilePath] = state.LogEnabled
	}
	for path, editor := range w.OpenEditors {
		if logEnabled, ok := logStateMap[path]; ok {
			editor.SetLogEnabled(logEnabled) // 应用保存的日志状态
		}
	}

	// 恢复活动文件
	if memento.ActiveFilePath != "" {
		if editor, ok := w.OpenEditors[memento.ActiveFilePath]; ok {
			w.SetActiveEditor(editor)
		}
	}
	return nil
}

// ------------------------------
// 核心业务方法（文件操作）
// ------------------------------

//现在恢复的文件的日志状态全部搞定，接下来就是新load的日志状态怎么初始化（根据文件头标识），另外如果
//原本没有log on 中途打开了，那要在文件的开头写入# log标记，下次加载默认是打开log
//原本没有log。打开后，先给编辑器加上标记，再给备忘录对象更新标记（备忘了不需要更新，因为他本身就是保存的时候动态读取编辑器的状态）
//所有的日志状态标记，维护在编辑器就够了
//在log onoff 开关的时候，只对编辑器修改，并且处理相关文件的首行，其他情况下，比如新建，读备忘区，都已经处理过

// LoadFile 功能：1. 拼接路径为 ./files/文件名 2. 检查文件是否已打开 3. 通过工厂创建编辑器 4. 加入工作区并设为激活
func (w *Workspace) LoadFile(path string, editorFactory func(path string, w common.WorkSpaceApi) (common.Editor, error)) (common.Editor, error) {
	// 1. 标准化路径：使用 filepath 包拼接，保证跨平台兼容性（Linux/macOS /, Windows \）
	// 仅拼接一次 ./files 目录，解决路径重复问题
	fullPath := filepath.Join("./files", path)

	if w.GetActiveEditor().GetFilePath() != "" {
		w.NotifyObservers(common.WorkspaceEvent{
			FilePath:  w.GetActiveEditor().GetFilePath(),
			Type:      common.EventFileSwitched,
			Command:   "",
			Timestamp: time.Now().UnixMilli(),
		})
	}

	// 2. 检查文件是否已在工作区中打开
	if editor, ok := w.OpenEditors[fullPath]; ok {
		w.SetActiveEditor(editor)
		firstLine := strings.Split(editor.GetContent(), "\n")[0]
		fmt.Println("first", firstLine)

		if strings.Contains(firstLine, "# log") {
			fmt.Println("ls")
			w.NotifyObservers(common.WorkspaceEvent{
				FilePath:  fullPath,
				Type:      "load",
				Timestamp: time.Now().UnixMilli(),
				Command:   "load " + fullPath,
			})
		}
		return editor, nil
	}

	// 3. 确保 ./files 目录存在（首次加载文件时创建，避免文件写入失败）
	if err := os.MkdirAll("./files", 0755); err != nil {
		return nil, errors.New("创建 files 目录失败: " + err.Error())
	}

	// 4. 通过工厂方法创建对应类型的编辑器（文本/XML）
	// 工厂方法接收的是完整路径 fullPath，可处理「当前目录下非工作区文件」的查找逻辑
	editor, err := editorFactory(fullPath, w)
	if err != nil {
		return nil, errors.New("创建编辑器失败: " + err.Error())
	}

	w.OpenEditors[fullPath] = editor
	w.SetActiveEditor(editor)
	//load指令日志
	firstLine := strings.Split(editor.GetContent(), "\n")[0]
	if strings.Contains(firstLine, "# log") {
		w.NotifyObservers(common.WorkspaceEvent{
			FilePath:  fullPath,
			Type:      "load",
			Timestamp: time.Now().UnixMilli(),
			Command:   "load " + fullPath,
		})
	}
	// 可选：通知观察者文件已加载（取消注释启用）
	// w.notifyObservers(common.WorkspaceEvent{
	// 	Type:      "FILE_LOADED",
	// 	Data:      map[string]string{"path": fullPath},
	// 	Timestamp: time.Now().UnixMilli(),
	// })

	return editor, nil
}

// SaveFile 保存文件
// 功能：1. 校验编辑器非空 2. 获取文件完整路径 3. 写入文件内容 4. 清除修改标记 5. 通知观察者
func (w *Workspace) SaveFile(editor common.Editor) error {
	// 1. 校验编辑器实例非空
	if editor == nil {
		return errors.New("editor is nil: 编辑器实例为空")
	}

	// 2. 获取编辑器中的完整文件路径（已在 LoadFile 中拼接为 ./files/文件名，无需再次拼接）
	path := editor.GetFilePath()
	if path == "" {
		return errors.New("file path is empty: 编辑器文件路径为空")
	}

	// 3. 确保文件所在目录存在（防止目录被手动删除后保存失败）
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.New("创建文件目录失败: " + err.Error())
	}

	// 4. 从编辑器中获取内容并写入文件
	content := editor.GetContent()
	//这里保存的时候，绝对不能从lines里保存，不然新增的变化，无法持久化
	//必须把树反序列化
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return errors.New("写入文件内容失败: " + err.Error())
	}

	// 5. 清除编辑器的修改标记
	editor.MarkAsModified(false)

	// 6. 若开启日志，通知观察者保存事件
	if editor.IsLogEnabled() {
		w.NotifyObservers(common.WorkspaceEvent{
			FilePath:  path,
			Type:      "Save",
			Command:   "Save " + path,
			Timestamp: time.Now().UnixMilli(),
		})
	}

	return nil
}

// CloseFile 关闭文件
func (w *Workspace) CloseFile(path string) error {

	if path == "" {
		return errors.New("file path is empty: 文件路径不能为空")
	}

	fullPath := filepath.Join("./files", path)

	if _, ok := w.OpenEditors[fullPath]; !ok {
		return errors.New("file not open: 文件未打开（查找路径：" + fullPath + "）")
	}

	editor := w.OpenEditors[fullPath]

	if editor.IsLogEnabled() {
		w.NotifyObservers(common.WorkspaceEvent{
			FilePath:  editor.GetFilePath(),
			Type:      "Close",
			Command:   "Close " + editor.GetFilePath(),
			Timestamp: time.Now().UnixMilli(),
		})
	}
	w.NotifyObservers(common.WorkspaceEvent{
		FilePath:  fullPath,
		Type:      common.EventFileClosed,
		Command:   "",
		Timestamp: time.Now().UnixMilli(),
	})
	delete(w.OpenEditors, fullPath)

	if w.activeEditor != nil && w.activeEditor.GetFilePath() == fullPath {
		// 7.1 若还有其他打开的文件，取第一个作为新的激活文件
		if len(w.OpenEditors) > 0 {
			for _, ed := range w.OpenEditors {
				w.SetActiveEditor(ed)
				break
			}
		} else {

			w.activeEditor = nil
		}
	}

	return nil
}

// SetActiveEditor 设置当前活动编辑器
func (w *Workspace) SetActiveEditor(editor common.Editor) {
	if editor == nil {
		return
	}
	//// 检查编辑器是否在工作区中
	//if _, ok := w.openEditors[editor.GetFilePath()]; !ok {
	//	return
	//}
	path := editor.GetFilePath()
	// 检查编辑器是否在工作区（已保存或未保存）
	if _, ok := w.OpenEditors[path]; !ok {
		return
	}
	w.activeEditor = editor
	w.NotifyObservers(common.WorkspaceEvent{
		FilePath:  path,
		Type:      common.EventFileActivated,
		Command:   "SetActiveEditor",
		Timestamp: time.Now().UnixMilli(),
	})
}

// // ToggleLog 切换日志开关状态
// func (w *Workspace) ToggleLog(enabled bool) {

// }

// GetActiveEditor 获取当前活动编辑器
func (w *Workspace) GetActiveEditor() common.Editor {
	return w.activeEditor
}

// GetOpenEditors 获取所有已打开的编辑器
func (w *Workspace) GetOpenEditors() []common.Editor {
	editors := make([]common.Editor, 0, len(w.OpenEditors))
	for _, editor := range w.OpenEditors {
		editors = append(editors, editor)
	}

	return editors
}
