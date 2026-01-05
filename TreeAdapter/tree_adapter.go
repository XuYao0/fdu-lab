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
	// 1. 确定当前节点的连接符
	symbol := "├── "
	if isLast {
		symbol = "└── "
	}

	// 2. 根节点特殊处理：不打印前缀和符号，只打印名字
	if prefix == "" {
		fmt.Println(node.Name)
	} else {
		fmt.Println(prefix + symbol + node.Name)
	}

	// 3. 计算给子节点使用的新前缀
	var newPrefix string
	if prefix == "" {
		// 根节点的子节点不需要前缀，但后续层级需要
		newPrefix = ""
	} else {
		if isLast {
			newPrefix = prefix + "    " // 后面没东西了，留空
		} else {
			newPrefix = prefix + "│   " // 后面还有兄弟节点，画竖线
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
		// 关键修复：根节点下的第一层节点 prefix 应该传 ""，但 isLast 要根据实际情况
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
	RootPath string // 文件目录的根路径
}

func (f *FileTreeAdapter) GetRootNode() *TreeNode {
	// 获取绝对路径后的最后一部分
	absPath, _ := filepath.Abs(f.RootPath)
	rootName := filepath.Base(absPath)

	// 如果 RootPath 是 "."，Base 会返回 "."，这没问题
	return &TreeNode{
		Name: rootName,
		Data: absPath, // 存绝对路径确保读取不出错
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
			Name: entry.Name(),                          // 必须只是文件名，例如 "apple.txt"
			Data: filepath.Join(nodePath, entry.Name()), // 必须是全路径
		})
	}
	return children
}

// XMLNode 定义XML节点的结构体（根据你的XML结构调整字段）
//
//	type XMLNode struct {
//		XMLName  xml.Name
//		Content  string     `xml:",chardata"`
//		Attrs    []xml.Attr `xml:",attr"`
//		Children []XMLNode  `xml:",any"` // 子节点
//	}
type XMLNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Content  string     `xml:",chardata"`
	Children []XMLNode  `xml:",any"`
}

// XMLTreeAdapter XML结构适配器
type XMLTreeAdapter struct {
	RootXML XMLNode // XML根节点
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
		// 1过滤掉 XML 解析器产生的空白字符节点（Local 为空说明不是真正的标签）
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
