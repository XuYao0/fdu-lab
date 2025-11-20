package storage

import (
	"encoding/json"
	"lab1/workspace"
	"os"
)

// LocalStorage 本地存储管理器（备忘录模式的Caretaker）
type LocalStorage struct {
	path string // 存储文件路径（如workspace_state.json）
}

// NewLocalStorage 创建本地存储实例
func NewLocalStorage(path string) *LocalStorage {
	return &LocalStorage{path: path}
}

// LoadMemento 从本地文件加载工作区备忘录
func (ls *LocalStorage) LoadMemento() (*workspace.WorkspaceMemento, error) {
	// 打开存储文件
	file, err := os.Open(ls.path)
	if err != nil {
		if os.IsNotExist(err) { // 文件不存在，返回空备忘录
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	// 解析JSON为Memento
	var memento workspace.WorkspaceMemento
	if err := json.NewDecoder(file).Decode(&memento); err != nil {
		return nil, err
	}
	return &memento, nil
}

// SaveMemento 将工作区备忘录保存到本地文件
func (ls *LocalStorage) SaveMemento(memento *workspace.WorkspaceMemento) error {
	// 创建存储文件（若不存在）
	file, err := os.Create(ls.path)
	if err != nil {
		return err
	}
	defer file.Close()

	// 将Memento序列化为JSON写入文件
	return json.NewEncoder(file).Encode(memento)
}
