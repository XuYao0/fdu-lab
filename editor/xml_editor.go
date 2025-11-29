package editor

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"lab1/common"
	"regexp"
	"strings"
)

type XmlEditorInterface interface {
	common.Editor
	InsertBefore(tag, newId, targetId, text string) error
	AppendChild(tag, newId, parentId, text string) error
	EditId(oldId, newId string) error
	EditText(elementId, text string) error
	Delete(elementId string) error
	XmlTree(filePath string) error
}

// XMLElement 具体元素节点（组合模式）
type XMLElement struct {
	tag      string            // 标签名
	id       string            // 唯一ID
	attrs    map[string]string // 属性集合
	text     string            // 文本内容
	parent   *XMLElement       // 父节点
	children []*XMLElement     // 子节点
}

// XmlEditor XML编辑器主结构：实现双接口
type XmlEditor struct {
	//editorType string
	filePath     string
	lines        []string
	root         *XMLElement            // 根元素
	idMap        map[string]*XMLElement // ID到元素的映射
	isModified   bool
	undoStack    []Command // 命令模式（Undo/Redo）
	redoStack    []Command
	logEnabled   bool
	logFilters   []string // 日志过滤命令列表
	workspaceApi common.WorkSpaceApi
}

func (x *XmlEditor) ExecuteCommand(command Command) {
	command.Execute()
	x.undoStack = append(x.undoStack, command)
	x.redoStack = nil
	x.isModified = true
}

func NewXmlEditor(path string, content string, wsApi common.WorkSpaceApi) *XmlEditor {
	editor := &XmlEditor{
		filePath:     path,
		lines:        strings.Split(content, "\n"), // 保留原始XML文本
		idMap:        make(map[string]*XMLElement),
		workspaceApi: wsApi,
		logEnabled:   false,
		isModified:   false,
	}
	firstLine := ""
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 {
		firstLine = strings.TrimSpace(lines[0])
	}
	logEnabled := strings.Contains(firstLine, "# log")
	if logEnabled {
		lines = lines[1:]
		content = strings.Join(lines, "\n")
	}
	// 关键：如果XML内容非空，自动解析为树形结构
	if content != "" {
		root, err := editor.parseXMLContent(content)
		if err != nil {
			fmt.Printf("XML解析警告：%s，将使用空根节点\n", err)
			// 解析失败时创建默认空根节点（避免nil）
			editor.root = &XMLElement{tag: "<解析失败>", id: "no-id", attrs: make(map[string]string)}
		} else {
			editor.root = root
			editor.buildIdMap(root) // 构建ID到节点的映射
		}
	} else {
		// 新文件（空内容）：创建默认根节点
		editor.root = &XMLElement{tag: "root", id: "root", attrs: map[string]string{"id": "root"}}
	}

	return editor
}

// parseXMLContent 解析XML文本为XMLElement树形结构
func (x *XmlEditor) parseXMLContent(content string) (*XMLElement, error) {
	// 去除XML注释（避免注释干扰解析）
	content = x.removeXMLComments(content)
	if content == "" {
		return nil, errors.New("XML内容为空")
	}

	decoder := xml.NewDecoder(strings.NewReader(content))
	var root *XMLElement
	var currentParent *XMLElement

	for {
		token, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break // 解析结束
			}
			return nil, fmt.Errorf("解析token失败：%w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			// 解析开始标签：创建节点并提取属性
			elem := &XMLElement{
				tag:      t.Name.Local,
				attrs:    make(map[string]string),
				parent:   currentParent,
				children: []*XMLElement{},
			}

			// 提取所有属性（包括id）
			for _, attr := range t.Attr {
				attrName := attr.Name.Local
				attrValue := attr.Value
				elem.attrs[attrName] = attrValue
				if attrName == "id" {
					elem.id = attrValue // 单独提取id属性
				}
			}

			// 初始化根节点
			if currentParent == nil {
				root = elem
			} else {
				// 非根节点：添加到父节点的子节点列表
				currentParent.children = append(currentParent.children, elem)
			}
			currentParent = elem // 进入子节点层级

		case xml.CharData:
			// 解析文本内容（去除首尾空格，空文本忽略）
			text := strings.TrimSpace(string(t))
			if text != "" && currentParent != nil {
				currentParent.text = text
			}

		case xml.EndElement:
			// 解析结束标签：回退到父节点
			if currentParent != nil {
				currentParent = currentParent.parent
			}
		}
	}

	if root == nil {
		return nil, errors.New("XML缺少根节点")
	}
	return root, nil
}

// buildIdMap 构建ID到节点的映射（方便通过ID快速查找节点）
func (x *XmlEditor) buildIdMap(root *XMLElement) {
	if root == nil {
		return
	}
	// 递归遍历所有节点
	var traverse func(*XMLElement)
	traverse = func(elem *XMLElement) {
		if elem.id != "" {
			x.idMap[elem.id] = elem
		}
		for _, child := range elem.children {
			traverse(child)
		}
	}
	traverse(root)
}

// removeXMLComments 去除XML中的注释（<!-- ... -->）
func (x *XmlEditor) removeXMLComments(content string) string {
	re := regexp.MustCompile(`<!--[\s\S]*?-->`)
	return re.ReplaceAllString(content, "")
}

func (x *XmlEditor) GetFilePath() string {
	return x.filePath
}

func (x *XmlEditor) IsModified() bool {
	return x.isModified
}
func (x *XmlEditor) MarkAsModified(modified bool) {
	x.isModified = modified
}

// GetTreeContent 这里不改，在保存的时候会把树形结构保存
// GetTreeContent GetContent 生成XML树形结构字符串（修复转义函数调用）
func (x *XmlEditor) GetTreeContent() string {
	// 空文档处理
	if x.root == nil {
		return "无XML内容（根节点未初始化）"
	}

	var buf strings.Builder
	// 写入标题（可选，增强可读性）
	buf.WriteString("XML树形结构:\n")
	// 递归生成根节点的树形结构（根节点无前置符号）
	x.buildTree(x.root, &buf, "", true)

	return buf.String()
}

// GetContent GetLinesContent 很重要！！！！！！！！！不然保存的时候会出问题
func (x *XmlEditor) GetContent() string {
	content, err := x.ToXML()
	if err != nil {
		fmt.Println(err)
	}
	if x.logEnabled {
		content = "# log\n" + content
	}
	return content
}

func (x *XmlEditor) ToXML() (string, error) {
	// 校验根节点是否为空
	if x.root == nil {
		return "", fmt.Errorf("XML根节点为空，无法序列化")
	}

	// 初始化缓冲区，用于拼接XML文本
	var buf bytes.Buffer

	// 写入XML声明（固定格式：<?xml version="1.0" encoding="UTF-8"?>）
	// xml.Header 是encoding/xml包提供的标准XML声明常量
	buf.WriteString(xml.Header)

	// 递归序列化根节点及其所有子节点（缩进为0级）
	if err := x.serializeNode(x.root, &buf, 0); err != nil {
		return "", fmt.Errorf("序列化节点失败: %w", err)
	}

	// 将缓冲区转换为字符串返回
	return buf.String(), nil
}

// serializeNode 递归序列化单个XMLElement节点为XML标签
// 参数：
//
//	elem: 待序列化的节点
//	buf: 用于拼接XML的缓冲区
//	indent: 当前节点的缩进级别（控制格式化的空格数）
//
// 返回：序列化过程中的错误信息
func (x *XmlEditor) serializeNode(elem *XMLElement, buf *bytes.Buffer, indent int) error {
	// 防御性校验：节点为空则直接返回
	if elem == nil {
		return nil
	}

	// 生成当前节点的缩进字符串（每级缩进4个空格，可自定义）
	indentStr := strings.Repeat("    ", indent)

	// 1. 写入开始标签的前缀（如：<bookstore）
	buf.WriteString(indentStr)
	buf.WriteString("<")
	buf.WriteString(elem.tag)

	// 2. 写入节点的所有属性（如：id="root"、category="COOKING"）
	// 遍历attrs映射，按XML语法拼接属性键值对
	for attrName, attrValue := range elem.attrs {
		// xml.EscapeString：对属性值进行XML转义（处理&、<、>、"、'等特殊字符）
		escapedValue := escapeXML(attrValue)
		buf.WriteString(fmt.Sprintf(` %s="%s"`, attrName, escapedValue))
	}

	// 3. 处理自闭合标签（无文本且无子节点的节点，如：<empty />）
	if elem.text == "" && len(elem.children) == 0 {
		buf.WriteString("/>\n")
		return nil
	}

	// 4. 闭合开始标签（如：<bookstore>）
	buf.WriteString(">\n")

	// 5. 写入节点的文本内容（若有）
	if elem.text != "" {
		// 文本内容的缩进级别比节点高1级
		textIndentStr := strings.Repeat("    ", indent+1)
		// 对文本内容进行XML转义
		escapedText := escapeXML(elem.text)
		buf.WriteString(textIndentStr)
		buf.WriteString(escapedText)
		buf.WriteString("\n")
	}

	// 6. 递归序列化当前节点的所有子节点
	for _, child := range elem.children {
		if err := x.serializeNode(child, buf, indent+1); err != nil {
			return err
		}
	}

	// 7. 写入结束标签（如：</bookstore>）
	buf.WriteString(indentStr)
	buf.WriteString("</")
	buf.WriteString(elem.tag)
	buf.WriteString(">\n")

	return nil
}

// buildTree 递归构建树形结构字符串（核心修复：替换为自定义escapeXML）
// buildTree 递归构建树形结构字符串（适配官方格式）
func (x *XmlEditor) buildTree(elem *XMLElement, buf *strings.Builder, prefix string, isLast bool) {
	if elem == nil {
		return
	}

	// 1. 拼接树形符号（非根节点）
	if elem.parent != nil {
		if isLast {
			buf.WriteString(prefix + "└── ")
		} else {
			buf.WriteString(prefix + "├── ")
		}
	}

	// 2. 拼接标签+所有属性（ID+其他，[]包裹）
	tag := elem.tag
	if tag == "" {
		tag = "<未知标签>"
	}
	buf.WriteString(tag)

	var attrs []string
	if elem.id != "" {
		attrs = append(attrs, fmt.Sprintf("id=\"%s\"", elem.id))
	}
	for k, v := range elem.attrs {
		if k == "id" {
			continue
		}
		attrs = append(attrs, fmt.Sprintf("%s=\"%s\"", k, escapeXML(v)))
	}
	if len(attrs) > 0 {
		buf.WriteString(" [" + strings.Join(attrs, ", ") + "]")
	}

	// 3. 处理文本内容：作为独立子节点展示
	if elem.text != "" {
		buf.WriteString("\n") // 文本节点前换行
		textPrefix := prefix
		if elem.parent != nil {
			if isLast {
				textPrefix += "    "
			} else {
				textPrefix += "│   "
			}
		}
		buf.WriteString(textPrefix + "└── \"" + escapeXML(elem.text) + "\"")
		buf.WriteString("\n") // 文本节点后强制换行
	}

	// 4. 核心修复：无文本但有子节点时，强制换行（关键）
	if len(elem.children) > 0 && elem.text == "" {
		buf.WriteString("\n")
	}

	// ========== 新增：空节点强制换行 ==========
	// 无文本且无子节点的空节点，强制添加换行，解决拼接问题
	if elem.text == "" && len(elem.children) == 0 {
		buf.WriteString("\n")
	}

	// 5. 处理子节点的前置符号（缩进）
	childPrefix := prefix
	if elem.parent != nil {
		if isLast {
			childPrefix += "    " // 最后一个子节点：用空格缩进
		} else {
			childPrefix += "│   " // 非最后一个子节点：用竖线保持层级
		}
	}

	// 6. 递归遍历子节点
	childCount := len(elem.children)
	for i := range elem.children {
		child := elem.children[i]
		// 标记当前子节点是否为最后一个
		childIsLast := (i == childCount-1)
		x.buildTree(child, buf, childPrefix, childIsLast)
	}
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")   // & 转义为 &amp;
	s = strings.ReplaceAll(s, "<", "&lt;")    // < 转义为 &lt;
	s = strings.ReplaceAll(s, ">", "&gt;")    // > 转义为 &gt;
	s = strings.ReplaceAll(s, "\"", "&quot;") // " 转义为 &quot;
	s = strings.ReplaceAll(s, "'", "&apos;")  // ' 转义为 &apos;
	return s
}

func (x *XmlEditor) Undo() error {
	if len(x.undoStack) == 0 {
		return nil
	}
	cmd := x.undoStack[len(x.undoStack)-1]
	cmd.Undo()
	x.undoStack = x.undoStack[:len(x.undoStack)-1]
	x.redoStack = append(x.redoStack, cmd)
	return nil
}

// Redo 重做操作
func (x *XmlEditor) Redo() error {
	if len(x.redoStack) == 0 {
		fmt.Println("redo stack is empty!")
		return nil
	}
	cmd := x.redoStack[len(x.redoStack)-1]
	cmd.Execute()
	x.redoStack = x.redoStack[:len(x.redoStack)-1]
	x.undoStack = append(x.undoStack, cmd)
	return nil
}

func (x *XmlEditor) SetLogEnabled(enabled bool) {
	// 1. 记录旧状态，若状态无变化则直接返回，避免无效操作
	oldEnabled := x.logEnabled
	x.logEnabled = enabled
	if oldEnabled == enabled {
		return
	}

	// 2. 根据开关状态，在内存中处理首行的# log标记
	if enabled {
		// 开启日志：首行无# log则插入（仅内存中）
		x.addLogMarkerInMemory()
	} else {
		// 关闭日志：首行有# log则移除（仅内存中）
		x.removeLogMarkerInMemory()
	}
}
func (x *XmlEditor) addLogMarkerInMemory() {
	if len(x.lines) == 0 {
		x.lines = []string{"# log"}
	} else {

		firstLine := strings.TrimSpace(x.lines[0])
		if firstLine != "# log" {

			x.lines = append([]string{"# log"}, x.lines...)
		}
	}
	// 标记文件为已修改（供后续持久化逻辑判断）
	x.MarkAsModified(true)
}

// removeLogMarkerInMemory 仅在内存中移除文件首行的# log标记（有则删）
func (x *XmlEditor) removeLogMarkerInMemory() {
	if len(x.lines) == 0 {
		return
	}

	// 去除首行空格后检查是否是目标标记
	firstLine := strings.TrimSpace(x.lines[0])
	if firstLine == "# log" {
		x.lines = x.lines[1:]
		x.MarkAsModified(true)
	}
}
func (x *XmlEditor) IsLogEnabled() bool {
	return x.logEnabled
}
