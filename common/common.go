package common

const (
	EventFileActivated string = "file_activated" // 文件成为活动文件
	EventFileSwitched  string = "file_switched"  // 切换到其他文件
	EventFileClosed    string = "file_closed"    // 文件被关闭
	EventProgramExit   string = "program_exit"   // 程序退出（可选）
)

// Editor 编辑器接口（文本编辑器、XML编辑器需实现）
type Editor interface {
	GetFilePath() string
	IsModified() bool
	MarkAsModified(modified bool)
	GetContent() string
	Undo() error
	Redo() error
	SetLogEnabled(a bool)
	IsLogEnabled() bool

	//

	//SpellCheck(checker SpellChecker) []SpellError
}

// WorkspaceEvent 工作区事件结构
type WorkspaceEvent struct {
	FilePath  string
	Type      string      // 事件类型：指令名
	Command   string      //原始指令本身
	Data      interface{} // 事件数据（根据类型不同而不同）
	Timestamp int64       // 事件发生时间戳
}

//type新增计时，只有对的上了才接受事件，日志模块先检查type

type Observer interface {
	Update(event WorkspaceEvent)
}

type WorkSpaceApi interface {
	NotifyObservers(event WorkspaceEvent)
}
