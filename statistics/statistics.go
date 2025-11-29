package statistics

import (
	"errors"
	"fmt"
	"lab1/common"
	"sync"
	"time"
)

//怎么和初始化适配，回复工作区？
//研究一下恢复能不能直接调用load

// FileTimer 单个文件的计时数据
type FileTimer struct {
	StartTime time.Time     // 最近一次激活的开始时间（未激活则为零值）
	TotalTime time.Duration // 累计编辑时长
	IsActive  bool          // 是否为当前活动文件
	IsClosed  bool          // 文件是否被关闭（关闭后重新打开需重置）
}

// Statistics 统计模块的核心结构体，实现Observer接口
type Statistics struct {
	timers map[string]*FileTimer // key: 文件路径，value: 计时数据
	mu     sync.RWMutex          // 并发安全锁（可选，单线程可移除）
	warnCh chan string           // 警告通道，用于输出统计失败信息
}

// NewStatistics 创建统计模块实例
func NewStatistics() *Statistics {
	return &Statistics{
		timers: make(map[string]*FileTimer),
		warnCh: make(chan string, 10), // 缓冲通道存储警告
	}
}

// Update 实现Observer接口，处理文件状态事件（核心计时逻辑）
func (s *Statistics) Update(state common.WorkspaceEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := state.FilePath
	switch state.Type {
	case common.EventFileActivated:
		// 处理文件激活：关闭则重置计时，否则开始新的计时
		if timer, ok := s.timers[filePath]; ok {
			if timer.IsClosed {
				// 关闭后重新打开，重置时长
				timer.TotalTime = 0
				timer.IsClosed = false
			}
			timer.StartTime = time.UnixMilli(state.Timestamp)
			timer.IsActive = true
		} else {
			// 首次激活，初始化计时数据
			s.timers[filePath] = &FileTimer{
				StartTime: time.UnixMilli(state.Timestamp),
				TotalTime: 0,
				IsActive:  true,
				IsClosed:  false,
			}
		}

	case common.EventFileSwitched, common.EventFileClosed:
		// 处理文件切换/关闭：停止计时并累计时长
		timer, ok := s.timers[filePath]
		if !ok || !timer.IsActive {
			return // 无计时数据或未激活，直接返回
		}
		// 计算本次激活的时长并累计
		elapsed := time.UnixMilli(state.Timestamp).Sub(timer.StartTime)
		if elapsed < 0 {
			s.warnCh <- fmt.Sprintf("统计警告：文件%s的计时时间为负，忽略本次时长", filePath)
			return
		}
		timer.TotalTime += elapsed
		timer.IsActive = false
		if state.Type == common.EventFileClosed {
			timer.IsClosed = true
		}

	case common.EventProgramExit:
		// 程序退出：对所有激活的文件强制停止计时
		for path, timer := range s.timers {
			if timer.IsActive {
				elapsed := time.UnixMilli(state.Timestamp).Sub(timer.StartTime)
				timer.TotalTime += elapsed
				timer.IsActive = false
				s.timers[path] = timer
			}
		}
	}
}

// GetFormattedDuration 获取文件的格式化时长（装饰器模式的核心方法）
func (s *Statistics) GetFormattedDuration(filePath string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	timer, ok := s.timers[filePath]
	if !ok {
		return "0秒" // 无计时数据，显示0秒
	}
	// 若文件仍在激活，计算实时时长
	total := timer.TotalTime
	if timer.IsActive {
		total += time.Since(timer.StartTime)
	}
	// 转换为可读格式
	return formatDuration(total)
}

// GetWarning 读取统计模块的警告信息（非阻塞）
func (s *Statistics) GetWarning() string {
	select {
	case warn := <-s.warnCh:
		return warn
	default:
		return ""
	}
}

// 私有方法：将time.Duration转换为可读格式（按规范实现）
func formatDuration(d time.Duration) string {
	// 转换为总秒数（向下取整）
	totalSeconds := int(d.Round(time.Second).Seconds())
	if totalSeconds < 0 {
		return "0秒"
	}

	// 定义时间单位换算
	secondsPerMinute := 60
	secondsPerHour := 60 * secondsPerMinute
	secondsPerDay := 24 * secondsPerHour

	// 按范围判断格式
	switch {
	case totalSeconds < secondsPerMinute:
		// <1分钟：X秒
		return fmt.Sprintf("%d秒", totalSeconds)
	case totalSeconds < secondsPerHour:
		// 1-59分钟：X分钟
		minutes := totalSeconds / secondsPerMinute
		return fmt.Sprintf("%d分钟", minutes)
	case totalSeconds < secondsPerDay:
		// 1-23小时：X小时Y分钟
		hours := totalSeconds / secondsPerHour
		minutes := (totalSeconds % secondsPerHour) / secondsPerMinute
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	default:
		// ≥24小时：X天Y小时
		days := totalSeconds / secondsPerDay
		hours := (totalSeconds % secondsPerDay) / secondsPerHour
		return fmt.Sprintf("%d天%d小时", days, hours)
	}
}

// ResetTimer 辅助方法：重置文件计时（可选，供外部调用）
func (s *Statistics) ResetTimer(filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.timers[filePath]; !ok {
		return errors.New("文件未被统计")
	}
	s.timers[filePath] = &FileTimer{
		StartTime: time.Time{},
		TotalTime: 0,
		IsActive:  false,
		IsClosed:  false,
	}
	return nil
}
