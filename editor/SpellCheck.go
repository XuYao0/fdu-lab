package editor

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
)

//
// ======================
// 统一的数据结构
// ======================
//

// ProxyItem 映射本地中转进程返回的 item
type ProxyItem struct {
	Offset      int      `json:"offset"`
	Length      int      `json:"length"`
	Suggestions []string `json:"suggestions"`
	Message     string   `json:"message"`
}

// ProxyResponse 中转服务的返回格式
type ProxyResponse struct {
	Items []ProxyItem `json:"items"`
}

//
// ======================
// SpellCheck 基础能力
// ======================
//

func SpellCheck(text string) (*ProxyResponse, error) {
	req := map[string]string{
		"text":     text,
		"language": "auto",
	}

	b, _ := json.Marshal(req)
	resp, err := http.Post("http://127.0.0.1:8089/spellcheck", "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data ProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

//
// ======================
// TXT 文件拼写检查
// ======================
//

// ConvertOffsetToLineCol 将 offset 转成行列
func ConvertOffsetToLineCol(text string, offset int) (row, col int) {
	row = 1
	col = 1
	for i := 0; i < offset && i < len(text); i++ {
		if text[i] == '\n' {
			row++
			col = 1
		} else {
			col++
		}
	}
	return
}

// SpellCheckTxt TXT 拼写检查入口
func SpellCheckTxt(text string) error {
	result, err := SpellCheck(text)
	if err != nil {
		return err
	}

	fmt.Println("拼写检查结果:")

	for _, it := range result.Items {
		row, col := ConvertOffsetToLineCol(text, it.Offset)
		word := text[it.Offset : it.Offset+it.Length]

		fmt.Printf("第%d行，第%d列: \"%s\" -> 建议: %v\n",
			row, col, word, it.Suggestions)
	}

	return nil
}

//
// ======================
// XML 文件拼写检查
// ======================
//

// XML 元素位置信息（备用）
type ElementInfo struct {
	Name   string
	Text   string
	Offset int
}

// SpellCheckXML XML 拼写检查入口
func SpellCheckXML(xmlContent string) error {
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	var stack []string

	fmt.Println("拼写检查结果:")

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch t := tok.(type) {

		case xml.StartElement:
			// push
			stack = append(stack, t.Name.Local)

		case xml.EndElement:
			// pop
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}

		case xml.CharData:
			raw := string(t)
			text := strings.TrimSpace(raw)

			if text == "" {
				continue
			}

			// 路径如 book/title
			path := strings.Join(stack, "/")

			// 英文分词
			words := splitEnglishWords(text)

			for _, w := range words {
				result, _ := SpellCheck(w)
				if len(result.Items) > 0 {
					fmt.Printf("元素 %s: \"%s\" -> 建议: %v\n",
						path, w, collectProxySuggestions(result))
				}
			}
		}
	}

	return nil
}

//
// ======================
// 工具函数
// ======================
//

// splitEnglishWords text → [words]
func splitEnglishWords(s string) []string {
	out := []string{}
	cur := strings.Builder{}

	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			cur.WriteRune(r)
		} else {
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// collectProxySuggestions 提取建议
func collectProxySuggestions(result *ProxyResponse) []string {
	s := []string{}
	for _, i := range result.Items {
		s = append(s, i.Suggestions...)
	}
	return s
}
