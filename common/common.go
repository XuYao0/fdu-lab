package common

// Editor 编辑器接口（文本编辑器、XML编辑器需实现）
type Editor interface {
	GetFilePath() string
	IsModified() bool
	MarkAsModified(modified bool)
	GetContent() string
	Undo() error
	Redo() error
	Show(startLine, endLine int)
	Append(content string)
	Insert(line, col int, text string)
	Delete(line, col, length int)
	Replace(line, col, length int, text string)
	SetLogEnabled(a bool)
	IsLogEnabled() bool
}

// WorkspaceEvent 工作区事件结构
type WorkspaceEvent struct {
	FilePath string 
	Type      string      // 事件类型：指令名
	Command   string //原始指令本身
	Data      interface{} // 事件数据（根据类型不同而不同）
	Timestamp int64       // 事件发生时间戳
}

type Observer interface {
	Update(event WorkspaceEvent)
}


type WorkSpaceApi interface{
	NotifyObservers(event WorkspaceEvent)
}