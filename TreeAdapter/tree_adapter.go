package TreeAdapter

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TreeDataProvider 定义树形结构的核心接口，所有适配者都要实现
type TreeDataProvider interface {
	GetRootNode() *TreeNode
	GetChildren(node *TreeNode) []*TreeNode
}

// TreeNode 通用树形节点，适配所有结构的统一节点格式
type TreeNode struct {
	Name     string
	Data     interface{}
	Children []*TreeNode
}

// PrintTree 通用树形打印函数（控制台版本，用制表符缩进）
// provider: 适配后的树形数据提供器
// node: 当前打印的节点
// prefix: 缩进前缀
// isLast: 是否是最后一个子节点
func PrintTree(provider TreeDataProvider, node *TreeNode, prefix string, isLast bool) {

	symbol := "├── "
	if isLast {
		symbol = "└── "
	}

	if prefix == "" {
		fmt.Println(node.Name)
	} else {
		fmt.Println(prefix + symbol + node.Name)
	}

	var newPrefix string
	if prefix == "" {

		newPrefix = ""
	} else {
		if isLast {
			newPrefix = prefix + "    "
		} else {
			newPrefix = prefix + "│   "
		}
	}

	// 如果是根节点，我们希望第一层子节点能有缩进感
	// 所以如果当前是根节点，给子节点的基础前缀设为空，但下一层逻辑会补上
	actualNextPrefix := newPrefix
	if prefix == "" {
		actualNextPrefix = ""
	}

	children := provider.GetChildren(node)
	for i, child := range children {
		// 根节点下的第一层节点 prefix 应该传 ""，但 isLast 要根据实际情况
		// 这样递归进去后，第一层子节点会显示 ├── 或 └──
		if prefix == "" {
			PrintTree(provider, child, " ", i == len(children)-1)
		} else {
			PrintTree(provider, child, actualNextPrefix, i == len(children)-1)
		}
	}
}

// FileTreeAdapter 文件目录适配器，适配文件系统结构
type FileTreeAdapter struct {
	RootPath string
}

func (f *FileTreeAdapter) GetRootNode() *TreeNode {
	// 获取绝对路径后的最后一部分
	absPath, _ := filepath.Abs(f.RootPath)
	rootName := filepath.Base(absPath)

	return &TreeNode{
		Name: rootName,
		Data: absPath,
	}
}

func (f *FileTreeAdapter) GetChildren(node *TreeNode) []*TreeNode {
	nodePath, ok := node.Data.(string)
	if !ok {
		return nil
	}

	entries, err := os.ReadDir(nodePath)
	if err != nil {
		return nil
	}

	var children []*TreeNode
	for _, entry := range entries {
		children = append(children, &TreeNode{
			Name: entry.Name(),
			Data: filepath.Join(nodePath, entry.Name()),
		})
	}
	return children
}

type XMLNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Content  string     `xml:",chardata"`
	Children []XMLNode  `xml:",any"`
}

// XMLTreeAdapter XML结构适配器
type XMLTreeAdapter struct {
	RootXML XMLNode
}

// GetRootNode 获取XML根节点
func (x *XMLTreeAdapter) GetRootNode() *TreeNode {
	rootName := x.RootXML.XMLName.Local
	return &TreeNode{
		Name: rootName,
		Data: x.RootXML,
	}
}

// GetChildren 获取XML节点的子节点
func (x *XMLTreeAdapter) GetChildren(node *TreeNode) []*TreeNode {
	xmlNode, ok := node.Data.(XMLNode)
	if !ok {
		return nil
	}

	var children []*TreeNode
	for _, childXML := range xmlNode.Children {

		if childXML.XMLName.Local == "" {
			continue
		}

		//  处理属性
		attrStr := ""
		if len(childXML.Attrs) > 0 {
			var attrs []string
			for _, attr := range childXML.Attrs {
				attrs = append(attrs, fmt.Sprintf("%s=\"%s\"", attr.Name.Local, attr.Value))
			}
			attrStr = fmt.Sprintf(" [%s]", strings.Join(attrs, ", "))
		}

		// 使用 strings.TrimSpace 清理掉换行符和多余空格
		content := strings.TrimSpace(childXML.Content)
		if content != "" {
			content = ": " + content
		}

		// 标签名 + 属性 + 内容
		// 结果示例: book [id="myBook"]: 我的自传
		fullNodeName := childXML.XMLName.Local + attrStr + content

		childNode := &TreeNode{
			Name: fullNodeName,
			Data: childXML,
		}
		children = append(children, childNode)
	}
	return children
}
