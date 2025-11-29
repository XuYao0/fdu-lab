package editor

import (
	"fmt"
	"strings"
)

// ------------------------------
// 1. 命令接口定义（命令模式核心）
// ------------------------------

// ------------------------------
// 2. AppendCommand：处理 "append" 命令（追加一行）
// ------------------------------

type AppendCommand struct {
	editor    *TextEditor // 关联的编辑器
	text      string      // 要追加的文本（整行）
	prevLines []string    // 追加前的所有行（用于撤销）
	executed  bool        // 是否执行成功
}

func NewAppendCommand(editor *TextEditor, text string) *AppendCommand {
	return &AppendCommand{
		editor: editor,
		text:   text,
	}
}

// 执行：在文件末尾追加一行

func (cmd *AppendCommand) Execute() {
	if cmd.editor == nil {
		return
	}

	// 保存当前状态（用于撤销）
	cmd.prevLines = make([]string, len(cmd.editor.lines))
	copy(cmd.prevLines, cmd.editor.lines)

	// 执行追加（新增一行）
	cmd.editor.lines = append(cmd.editor.lines, cmd.text)
	cmd.editor.isModified = true
	cmd.executed = true

	// 触发事件（供观察者如日志模块使用）
	//cmd.editor.notifyEvent(Event{
	//	Type: "append",
	//	Data: map[string]interface{}{"text": cmd.text, "line": len(cmd.editor.lines)},
	//	Time: time.Now().UnixMilli(),
	//})

}

// 撤销：删除最后一行（恢复到追加前）

func (cmd *AppendCommand) Undo() {
	if !cmd.executed || cmd.editor == nil {
		return
	}

	// 恢复到追加前的行状态
	cmd.editor.lines = cmd.prevLines
	cmd.editor.isModified = true
}

func (cmd *AppendCommand) IsExecuted() bool {
	return cmd.executed
}

// ------------------------------
// 3. InsertCommand：处理 "insert" 命令（指定位置插入，支持换行）
// ------------------------------

type InsertCommand struct {
	editor     *TextEditor // 关联的编辑器
	line       int         // 目标行号（1-based）
	col        int         // 目标列号（1-based）
	text       string      // 插入的文本（可能含换行符）
	prevLine   string      // 插入前的目标行内容（用于撤销）
	splitLines []string    // 文本按换行拆分后的行（用于执行）
	executed   bool        // 是否执行成功
}

func NewInsertCommand(editor *TextEditor, line, col int, text string) *InsertCommand {
	return &InsertCommand{
		editor: editor,
		line:   line,
		col:    col,
		text:   text,
	}
}

// 执行：在指定位置插入文本（支持换行拆分）

func (cmd *InsertCommand) Execute() {
	if cmd.editor == nil || !cmd.validate() {
		return
	}

	// 转换为 0-based 索引
	lineIdx := cmd.line - 1
	colIdx := cmd.col - 1

	// 保存插入前的行内容（用于撤销）
	cmd.prevLine = cmd.editor.lines[lineIdx]

	// 按换行符拆分文本（支持多行插入）
	cmd.splitLines = strings.Split(cmd.text, "\n")

	// 执行插入逻辑
	if len(cmd.splitLines) == 1 {
		// 无换行：直接插入到当前行
		currentLine := cmd.prevLine
		newLine := currentLine[:colIdx] + cmd.text + currentLine[colIdx:]
		cmd.editor.lines[lineIdx] = newLine
	} else {
		// 有换行：拆分当前行并插入多行
		currentLine := cmd.prevLine
		// 第一部分：当前行从开始到插入位置 + 拆分的第一行
		firstPart := currentLine[:colIdx] + cmd.splitLines[0]
		// 中间部分：拆分的中间行（除首尾外）
		middleParts := cmd.splitLines[1 : len(cmd.splitLines)-1]
		// 最后部分：拆分的最后一行 + 当前行从插入位置到结尾
		lastPart := cmd.splitLines[len(cmd.splitLines)-1] + currentLine[colIdx:]

		// 重组所有行（插入新行）
		newLines := make([]string, 0, len(cmd.editor.lines)+len(middleParts)+1)
		newLines = append(newLines, cmd.editor.lines[:lineIdx]...)   // 插入行之前的内容
		newLines = append(newLines, firstPart)                       // 第一部分
		newLines = append(newLines, middleParts...)                  // 中间部分
		newLines = append(newLines, lastPart)                        // 最后部分
		newLines = append(newLines, cmd.editor.lines[lineIdx+1:]...) // 插入行之后的内容
		cmd.editor.lines = newLines
	}

	cmd.editor.isModified = true
	cmd.executed = true

	// 触发事件
	//cmd.editor.notifyEvent(Event{
	//	Type: "insert",
	//	Data: map[string]interface{}{"line": cmd.line, "col": cmd.col, "text": cmd.text},
	//	Time: time.Now().UnixMilli(),
	//})
}

// 撤销：移除插入的内容（恢复到插入前）

func (cmd *InsertCommand) Undo() {
	if !cmd.executed || cmd.editor == nil {
		return
	}

	lineIdx := cmd.line - 1

	if len(cmd.splitLines) == 1 {
		// 无换行：直接恢复原行
		cmd.editor.lines[lineIdx] = cmd.prevLine
	} else {
		// 有换行：合并被拆分的行，删除插入的中间行
		removeCount := len(cmd.splitLines) - 1 // 需要删除的行数
		newLines := make([]string, 0, len(cmd.editor.lines)-removeCount)
		newLines = append(newLines, cmd.editor.lines[:lineIdx]...)               // 插入行之前的内容
		newLines = append(newLines, cmd.prevLine)                                // 恢复原行
		newLines = append(newLines, cmd.editor.lines[lineIdx+1+removeCount:]...) // 跳过插入的中间行
		cmd.editor.lines = newLines
	}

	cmd.editor.isModified = true
}

// 验证插入位置是否合法
func (cmd *InsertCommand) validate() bool {
	lineCount := len(cmd.editor.lines)

	// 空文件只能在 1:1 位置插入
	if lineCount == 0 {
		return cmd.line == 1 && cmd.col == 1
	}

	// 行号越界（必须在 1~lineCount 之间）
	if cmd.line < 1 || cmd.line > lineCount {
		return false
	}

	// 列号越界（必须在 1~行长度+1 之间，允许插入到 行尾）

	targetLine := cmd.editor.lines[cmd.line-1]
	return cmd.col >= 1 && cmd.col <= len(targetLine)+1
}

func (cmd *InsertCommand) IsExecuted() bool {
	return cmd.executed
}

// ------------------------------
// 4. DeleteCommand：处理 "delete" 命令（删除指定长度字符）
// ------------------------------

type DeleteCommand struct {
	editor   *TextEditor // 关联的编辑器
	line     int         // 目标行号（1-based）
	col      int         // 起始列号（1-based）
	length   int         // 删除长度
	prevLine string      // 删除前的行内容（用于撤销）
	executed bool        // 是否执行成功
}

func NewDeleteCommand(editor *TextEditor, line, col, length int) *DeleteCommand {
	return &DeleteCommand{
		editor: editor,
		line:   line,
		col:    col,
		length: length,
	}
}

// 执行：删除指定范围的字符（不可跨行）

func (cmd *DeleteCommand) Execute() {
	if cmd.editor == nil || !cmd.validate() {
		return
	}

	lineIdx := cmd.line - 1
	colIdx := cmd.col - 1

	// 保存删除前的行内容（用于撤销）
	cmd.prevLine = cmd.editor.lines[lineIdx]

	// 执行删除
	currentLine := cmd.prevLine
	newLine := currentLine[:colIdx] + currentLine[colIdx+cmd.length:]
	cmd.editor.lines[lineIdx] = newLine

	cmd.editor.isModified = true
	cmd.executed = true

	// 触发事件
	//cmd.editor.notifyEvent(Event{
	//	Type: "delete",
	//	Data: map[string]interface{}{"line": cmd.line, "col": cmd.col, "length": cmd.length},
	//	Time: time.Now().UnixMilli(),
	//})
}

// 撤销：恢复被删除的字符

func (cmd *DeleteCommand) Undo() {
	if !cmd.executed || cmd.editor == nil {
		return
	}

	// 恢复原行内容
	cmd.editor.lines[cmd.line-1] = cmd.prevLine
	cmd.editor.isModified = true
}

// 验证删除范围是否合法
func (cmd *DeleteCommand) validate() bool {
	lineCount := len(cmd.editor.lines)

	// 行号越界
	if cmd.line < 1 || cmd.line > lineCount {
		return false
	}

	targetLine := cmd.editor.lines[cmd.line-1]
	lineLen := len(targetLine)
	colIdx := cmd.col - 1

	// 列号越界或删除长度无效
	if colIdx < 0 || colIdx >= lineLen || cmd.length <= 0 {
		return false
	}

	// 删除范围不能超过行尾
	if colIdx+cmd.length > lineLen {
		return false
	}

	return true
}

func (cmd *DeleteCommand) IsExecuted() bool {
	return cmd.executed
}

// ------------------------------
// 5. ReplaceCommand：处理 "replace" 命令（先删后插）
// ------------------------------

type ReplaceCommand struct {
	editor    *TextEditor    // 关联的编辑器
	line      int            // 目标行号（1-based）
	col       int            // 起始列号（1-based）
	length    int            // 删除长度
	text      string         // 替换的新文本
	deleteCmd *DeleteCommand // 内部删除命令
	insertCmd *InsertCommand // 内部插入命令
	executed  bool           // 是否执行成功
}

func NewReplaceCommand(editor *TextEditor, line, col, length int, text string) *ReplaceCommand {
	return &ReplaceCommand{
		editor:    editor,
		line:      line,
		col:       col,
		length:    length,
		text:      text,
		deleteCmd: NewDeleteCommand(editor, line, col, length),
		insertCmd: NewInsertCommand(editor, line, col, text), // 插入位置与删除位置相同
	}
}

// 执行：先删除指定长度字符，再插入新文本

func (cmd *ReplaceCommand) Execute() {
	if cmd.editor == nil {
		return
	}

	// 先执行删除
	cmd.deleteCmd.Execute()
	if !cmd.deleteCmd.IsExecuted() {
		return // 删除失败则终止替换
	}

	// 再执行插入（删除后行结构可能变化，但插入位置仍基于原行号）
	cmd.insertCmd.Execute()
	cmd.executed = cmd.insertCmd.IsExecuted()

}

// 撤销：先撤销插入，再撤销删除（恢复原状态）

func (cmd *ReplaceCommand) Undo() {
	if !cmd.executed || cmd.editor == nil {
		return
	}

	// 先撤销插入（移除新文本）
	cmd.insertCmd.Undo()
	// 再撤销删除（恢复原文本）
	cmd.deleteCmd.Undo()

	cmd.editor.isModified = true
}
func (cmd *ReplaceCommand) IsExecuted() bool {
	return cmd.executed
}

// --------------------------
// 1. XML命令结构体定义（命令模式）
// --------------------------

// InsertBeforeCommand 插入元素到目标元素前的命令
type InsertBeforeCommand struct {
	editor   *XmlEditor
	tag      string
	newId    string
	targetId string
	text     string
	// 用于撤销的临时存储
	insertedElem *XMLElement
	success      bool
}

// AppendChildCommand 追加子元素到父元素的命令
type AppendChildCommand struct {
	editor       *XmlEditor
	tag          string
	newId        string
	parentId     string
	text         string
	insertedElem *XMLElement
	success      bool
}

// EditIdCommand 修改元素ID的命令
type EditIdCommand struct {
	editor *XmlEditor
	oldId  string
	newId  string
	// 用于撤销的原ID
	prevId  string
	success bool
}

// EditTextCommand 修改元素文本的命令
type EditTextCommand struct {
	editor    *XmlEditor
	elementId string
	text      string
	// 用于撤销的原文本
	prevText string
	success  bool
}

// XmlDeleteCommand 删除元素的命令

type XmlDeleteCommand struct {
	editor      *XmlEditor
	elementId   string
	deletedElem *XMLElement
	parentElem  *XMLElement
	index       int
	success     bool
	// 新增：保存所有被删除节点的ID映射（用于撤销恢复）
	deletedIdMap map[string]*XMLElement
}

func NewInsertBeforeCommand(editor *XmlEditor, tag, newId, targetId, text string) *InsertBeforeCommand {
	return &InsertBeforeCommand{
		editor:   editor,
		tag:      tag,
		newId:    newId,
		targetId: targetId,
		text:     text,
	}
}

func (c *InsertBeforeCommand) Execute() {
	// 异常1：newId已存在
	if _, ok := c.editor.idMap[c.newId]; ok {
		fmt.Printf("元素ID已存在: %s\n", c.newId)
		return
	}

	// 异常2：targetId不存在
	targetElem, ok := c.editor.idMap[c.targetId]
	if !ok {
		fmt.Printf("目标元素不存在: %s\n", c.targetId)
		return
	}

	// 异常3：尝试在根元素前插入（根元素的parent为nil）
	parent := targetElem.parent
	if parent == nil {
		fmt.Println("不能在根元素前插入元素")
		return
	}

	// 找到目标元素在父节点子列表中的索引
	index := -1
	for i, child := range parent.children {
		if child.id == c.targetId {
			index = i
			break
		}
	}
	// 兜底：目标元素不在父节点的子列表中（理论上不会触发，因idMap已校验）
	if index == -1 {
		fmt.Printf("目标元素不存在: %s\n", c.targetId)
		return
	}

	// 创建新元素
	newElem := &XMLElement{
		tag:    c.tag,
		id:     c.newId,
		text:   c.text,
		attrs:  make(map[string]string),
		parent: parent,
	}
	// 同步id到attrs，保证序列化时生成id属性
	newElem.attrs["id"] = escapeXML(c.newId)

	// 插入新元素到目标元素前方（Go切片插入技巧）
	parent.children = append(parent.children[:index], append([]*XMLElement{newElem}, parent.children[index:]...)...)

	// 更新编辑器状态
	c.editor.idMap[c.newId] = newElem
	c.editor.isModified = true
	c.insertedElem = newElem
	c.success = true

	// 执行成功无需额外提示（按你的示例，仅异常提示）
}

func (c *InsertBeforeCommand) Undo() {
	if !c.success || c.insertedElem == nil {
		return
	}
	// 从父节点中删除插入的元素
	parent := c.insertedElem.parent
	if parent == nil {
		return
	}
	index := -1
	for i, child := range parent.children {
		if child.id == c.newId {
			index = i
			break
		}
	}
	if index != -1 {
		parent.children = append(parent.children[:index], parent.children[index+1:]...)
	}
	// 删除ID映射
	delete(c.editor.idMap, c.newId)
	c.editor.isModified = true
	c.success = false
	return
}

func (c *InsertBeforeCommand) IsExecuted() bool {
	return c.success
}

func NewAppendChildCommand(editor *XmlEditor, tag, newId, parentId, text string) *AppendChildCommand {
	return &AppendChildCommand{
		editor:   editor,
		tag:      tag,
		newId:    newId,
		parentId: parentId,
		text:     text,
	}
}

func (c *AppendChildCommand) Execute() {
	// 校验父元素是否存在
	parentElem, ok := c.editor.idMap[c.parentId]
	if !ok {
		// 兜底：递归遍历节点查找父节点（修复变量未定义问题）
		fmt.Printf("idMap中未找到父节点[%s]，尝试遍历节点查找...\n", c.parentId)
		// 使用c.editor.root作为根节点，调用findNodeById方法
		parentNode := c.editor.findNodeById(c.editor.root, c.parentId)
		if parentNode == nil {
			// 规范错误打印，让用户看到明确提示
			fmt.Printf("执行失败：parent not exist → 父节点ID [%s] 不存在（idMap和遍历均未找到）\n", c.parentId)
			c.success = false // 标记执行失败
			return
		}
		// 关键：将遍历找到的父节点赋值给parentElem
		parentElem = parentNode
		fmt.Printf("遍历节点成功找到父节点：%s(id=%s)\n", parentElem.tag, c.parentId)
	}

	// 校验新ID是否重复
	if _, ok := c.editor.idMap[c.newId]; ok {
		fmt.Println("old child")
		return
	}

	// 创建新元素
	newElem := &XMLElement{
		tag:    c.tag,
		id:     c.newId,
		text:   c.text,
		attrs:  make(map[string]string),
		parent: parentElem,
	}
	//这个不能少！！
	newElem.attrs["id"] = c.newId
	// 追加为子元素
	parentElem.children = append(parentElem.children, newElem)
	// 更新ID映射和状态
	c.editor.idMap[c.newId] = newElem
	c.editor.isModified = true
	c.insertedElem = newElem
	c.success = true
	return
}
func (x *XmlEditor) findNodeById(node *XMLElement, targetId string) *XMLElement {
	if node == nil {
		return nil
	}
	if node.id == targetId {
		return node
	}
	// 递归遍历子节点
	for _, child := range node.children {
		found := x.findNodeById(child, targetId)
		if found != nil {
			return found
		}
	}
	return nil
}
func (c *AppendChildCommand) Undo() {
	if !c.success || c.insertedElem == nil {
		return
	}
	// 从父节点中删除追加的元素
	parent := c.insertedElem.parent
	if parent == nil {
		return
	}
	index := -1
	for i, child := range parent.children {
		if child.id == c.newId {
			index = i
			break
		}
	}
	if index != -1 {
		parent.children = append(parent.children[:index], parent.children[index+1:]...)
	}
	// 删除ID映射
	delete(c.editor.idMap, c.newId)
	c.editor.isModified = true
	c.success = false
	return
}

func (c *AppendChildCommand) IsExecuted() bool {
	return c.success
}

func NewEditIdCommand(editor *XmlEditor, oldId, newId string) *EditIdCommand {
	return &EditIdCommand{
		editor: editor,
		oldId:  oldId,
		newId:  newId,
	}
}

func (c *EditIdCommand) Execute() {
	// 校验原元素是否存在
	elem, ok := c.editor.idMap[c.oldId]
	if !ok {
		fmt.Println("元素不存在:", c.oldId)
		return
	}
	if elem == c.editor.root {
		fmt.Println("【错误】禁止修改根节点的ID，操作已终止！")
		c.success = false
		return
	}
	// 校验新ID是否重复
	if _, ok := c.editor.idMap[c.newId]; ok {
		fmt.Println("目标ID已存在：", c.newId)
		return
	}
	// 保存原ID用于撤销
	c.prevId = elem.id
	// 修改元素ID
	delete(c.editor.idMap, c.oldId)
	elem.id = c.newId
	c.editor.idMap[c.newId] = elem
	c.editor.isModified = true
	c.success = true
	return
}

func (c *EditIdCommand) Undo() {
	if !c.success || c.prevId == "" {
		return
	}
	// 恢复原ID
	elem, ok := c.editor.idMap[c.newId]
	if !ok {
		return
	}
	delete(c.editor.idMap, c.newId)
	elem.id = c.prevId
	c.editor.idMap[c.prevId] = elem
	c.editor.isModified = true
	c.success = false
	return
}

func (c *EditIdCommand) IsExecuted() bool {
	return c.success
}

func NewEditTextCommand(editor *XmlEditor, elementId, text string) *EditTextCommand {
	return &EditTextCommand{
		editor:    editor,
		elementId: elementId,
		text:      text,
	}
}

func (c *EditTextCommand) Execute() {
	// 校验元素是否存在
	elem, ok := c.editor.idMap[c.elementId]
	if !ok {
		fmt.Println("元素不存在：", c.elementId)
		return
	}
	// 保存原文本用于撤销
	c.prevText = elem.text
	// 修改文本
	elem.text = c.text
	c.editor.isModified = true
	c.success = true
	return
}

func (c *EditTextCommand) Undo() {
	if !c.success || c.prevText == "" {
		return
	}
	// 恢复原文本
	elem, ok := c.editor.idMap[c.elementId]
	if !ok {
		return
	}
	elem.text = c.prevText
	c.editor.isModified = true
	c.success = false
	return
}

func (c *EditTextCommand) IsExecuted() bool {
	return c.success
}

func NewXmlDeleteCommand(editor *XmlEditor, elementId string) *XmlDeleteCommand {
	return &XmlDeleteCommand{
		editor:    editor,
		elementId: elementId,
	}
}

func (c *XmlDeleteCommand) Execute() {
	// 1. 校验元素是否存在
	elem, ok := c.editor.idMap[c.elementId]
	if !ok {
		fmt.Println("元素不存在：", c.elementId)
		return
	}

	// 2. 禁止删除根节点
	if elem == c.editor.root {
		fmt.Println("不能删除根元素")
		return
	}

	// ========== 关键修复：强制初始化deletedIdMap，避免为nil ==========
	c.deletedIdMap = make(map[string]*XMLElement)

	// 3. 递归保存被删除节点及其所有子节点到deletedIdMap
	c.recursiveSaveNodes(elem)
	fmt.Printf("[删除调试] 已保存 %d 个节点到deletedIdMap\n", len(c.deletedIdMap)) // 调试日志

	// 4. 保存删除的元素、父节点
	c.deletedElem = elem
	c.parentElem = elem.parent

	// 5. 查找节点在父节点中的索引（用内存地址对比更可靠）
	index := -1
	for i, child := range c.parentElem.children {
		if child == elem { // 优先通过内存地址对比
			index = i
			break
		}
	}
	c.index = index
	if index == -1 {
		fmt.Println("执行失败：未在父节点中找到该节点")
		c.success = false
		return
	}
	fmt.Printf("[删除调试] 节点%s的索引：%d\n", c.elementId, index)

	// 6. 从父节点中删除元素
	c.parentElem.children = append(c.parentElem.children[:index], c.parentElem.children[index+1:]...)

	// 7. 递归删除idMap中的映射
	c.recursiveDeleteIdMap(elem)

	// 8. 更新状态
	c.editor.isModified = true
	c.success = true
	return
}

// 递归保存节点及其所有子节点到deletedIdMap
func (c *XmlDeleteCommand) recursiveSaveNodes(elem *XMLElement) {
	if elem == nil || elem.id == "" {
		return
	}
	// 将当前节点存入map
	c.deletedIdMap[elem.id] = elem
	fmt.Printf("[删除调试] 保存节点%s到deletedIdMap\n", elem.id) // 调试日志

	// 递归保存子节点（price4无子女，此循环不会执行）
	for _, child := range elem.children {
		c.recursiveSaveNodes(child)
	}
}

// 递归删除节点及其所有子节点的ID映射
func (c *XmlDeleteCommand) recursiveDeleteIdMap(elem *XMLElement) {
	if elem == nil || elem.id == "" {
		return
	}
	// 先递归删除子节点（price4无子女，此循环不会执行）
	for _, child := range elem.children {
		c.recursiveDeleteIdMap(child)
	}
	// 删除当前节点
	delete(c.editor.idMap, elem.id)
	fmt.Printf("已从idMap中删除节点：%s\n", elem.id)
}

func (c *XmlDeleteCommand) Undo() {
	fmt.Println("[撤销调试] 执行XmlDeleteCommand的Undo方法")
	// 校验撤销的前置条件
	fmt.Printf("[撤销调试] success: %t\n", c.success)
	fmt.Printf("[撤销调试] deletedElem: %v (nil? %t)\n", c.deletedElem, c.deletedElem == nil)
	fmt.Printf("[撤销调试] parentElem: %v (nil? %t)\n", c.parentElem, c.parentElem == nil)
	fmt.Printf("[撤销调试] index: %d\n", c.index)
	fmt.Printf("[撤销调试] deletedIdMap: %v (nil? %t)\n", c.deletedIdMap, c.deletedIdMap == nil)

	if !c.success || c.deletedElem == nil || c.parentElem == nil || c.index == -1 || c.deletedIdMap == nil {
		fmt.Println("undo err")
		return
	}

	// 1. 恢复主节点到父节点的原位置
	c.parentElem.children = append(c.parentElem.children[:c.index], append([]*XMLElement{c.deletedElem}, c.parentElem.children[c.index:]...)...)

	// 2. 恢复所有被删除节点的ID映射（主节点+所有子节点）
	for id, elem := range c.deletedIdMap {
		c.editor.idMap[id] = elem
	}
	fmt.Printf("撤销删除：成功恢复 %d 个节点的ID映射\n", len(c.deletedIdMap))

	// 3. 标记编辑器为已修改，重置命令执行状态
	c.editor.isModified = true
	c.success = false

	// 可选：清空保存的子节点映射（避免重复撤销）
	// c.deletedChildrenMap = nil
	return
}

func (c *XmlDeleteCommand) IsExecuted() bool {
	return c.success
}
