package log

import (
	"bufio"
	"fmt"
	"lab1/common"
	"path/filepath"
	"strings"

	//"lab1/workspace"
	"os"
	"time"
)

// LogModule 日志模块（实现workspace.Observer接口）
type LogModule struct {
	logHandles   map[string]*os.File // 键：文件路径（如"a.txt"），值：对应日志文件句柄（.a.txt.log）
	sessionStart string              // 会话开始时间（用于日志头部）
}

// NewLogModule 创建日志模块实例
func NewLogModule() *LogModule {
	return &LogModule{
		logHandles:   make(map[string]*os.File), // 初始化句柄映射
		sessionStart: time.Now().Format("20060102 15:04:05"),
	}
}
func readFirstLine(handle *os.File) (string, error) {
	scanner := bufio.NewScanner(handle)
	if scanner.Scan() { // 读取第一行
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("文件为空")
}

// 获取或创建指定文件的日志句柄
// getLogHandle 按 ./logs/.文件名.log 格式生成日志文件，修复路径和目录问题
func (l *LogModule) getLogHandle(filePath string) (*os.File, error) {
	// 1. 复用已存在的日志句柄
	if handle, exists := l.logHandles[filePath]; exists {
		return handle, nil
	}

	// 2. 提取原文件的【基础文件名】（关键：去掉目录层级，只保留xxx.txt）
	// 示例1：filePath = "huawei.txt" → baseName = "huawei.txt"
	// 示例2：filePath = "files/shabi.txt" → baseName = "shabi.txt"
	baseName := filepath.Base(filePath)

	// 3. 按你的要求生成日志文件名：.基础文件名.log（如 .huawei.txt.log）
	logFileName := "." + baseName + ".log"

	// 4. 拼接日志文件的完整路径：./logs/.基础文件名.log（跨平台兼容）
	logDir := "./logs"
	logPath := filepath.Join(logDir, logFileName) // Windows下会自动转为 .\logs\.huawei.txt.log

	// 5. 提前创建 ./logs 目录（核心：解决目录不存在的报错）
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	// 6. 以追加模式打开/创建日志文件
	handle, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	// 7. 首次创建时写入会话开始标识
	if _, err := handle.WriteString("session start at " + l.sessionStart + "\n"); err != nil {
		_ = handle.Close() // 写入失败关闭句柄，避免泄露
		return nil, err
	}

	// 8. 缓存句柄
	l.logHandles[filePath] = handle
	return handle, nil
}

// Update 实现Observer接口：根据事件中的文件路径写入对应日志
func (l *LogModule) Update(event common.WorkspaceEvent) {
	// 从事件中提取文件路径和命令（假设事件结构按之前设计）
	Type := event.Type
	if Type == common.EventFileActivated || Type == common.EventFileSwitched || Type == common.EventFileClosed || Type == common.EventProgramExit {
		return
	}
	filePath := event.FilePath
	command := event.Command
	if filePath == "" || command == "" {
		return
	}

	// 获取该文件的日志句柄
	handle, err := l.getLogHandle(filePath)
	if err != nil {
		fmt.Printf("警告：无法打开日志文件（%s）：%v\n", "."+filePath+".log", err)
		return
	}

	_handle, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("无法打开或创建文件:", err)
		return
	}

	firstLine, err := readFirstLine(_handle)
	if err != nil {
		fmt.Println("read the first line failed:", err)
		return
	} //在这里加一个过滤功能，读取第一行，如果当前的Type 被禁止了，那直接返回
	if strings.Contains(firstLine, event.Type) {
		return
	}

	timeStr := time.UnixMilli(event.Timestamp).Format("20060102 15:04:05")
	logLine := fmt.Sprintf("%s %s\n", timeStr, command)

	// 写入日志
	if _, err := handle.WriteString(logLine); err != nil {
		fmt.Printf("警告：日志写入失败（%s）：%v\n", "."+filePath+".log", err)
	}
}

// Close 关闭所有日志句柄（程序退出时调用）
func (l *LogModule) Close() error {
	var lastErr error
	for _, handle := range l.logHandles {
		if err := handle.Close(); err != nil {
			lastErr = err // 记录最后一个错误
		}
	}
	return lastErr
}
