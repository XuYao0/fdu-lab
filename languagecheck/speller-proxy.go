package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const LTEndpoint = "https://api.languagetool.org/v2/check"

type CheckRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"`
}

type Match struct {
	Message      string `json:"message"`
	ShortMessage string `json:"shortMessage"`
	Offset       int    `json:"offset"`
	Length       int    `json:"length"`
	Rule         struct {
		ID   string `json:"id"`
		Desc string `json:"description"`
	} `json:"rule"`
	Replacements []struct {
		Value string `json:"value"`
	} `json:"replacements"`
}

type LTResponse struct {
	Matches []Match `json:"matches"`
}

type ProxyResponse struct {
	Items []ProxyItem `json:"items"`
}

type ProxyItem struct {
	Offset      int      `json:"offset"`
	Length      int      `json:"length"`
	Word        string   `json:"word"`
	Suggestions []string `json:"suggestions"`
	Message     string   `json:"message"`
}

func main() {
	http.HandleFunc("/spellcheck", handleSpellCheck)

	fmt.Println("SpellCheck proxy running on http://127.0.0.1:8089")
	log.Fatal(http.ListenAndServe("127.0.0.1:8089", nil))
}

func handleSpellCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Language == "" {
		req.Language = "auto"
	}

	// call LanguageTool
	resp, err := callLangTool(req.Text, req.Language)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func callLangTool(text, lang string) (*ProxyResponse, error) {
	body := fmt.Sprintf("text=%s&language=%s", text, lang)
	req, _ := http.NewRequest("POST", LTEndpoint, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var lt LTResponse
	if err = json.Unmarshal(raw, &lt); err != nil {
		return nil, err
	}

	// format output
	out := &ProxyResponse{}
	for _, m := range lt.Matches {
		item := ProxyItem{
			Offset:  m.Offset,
			Length:  m.Length,
			Message: m.Message,
		}

		if len(m.Replacements) > 0 {
			for _, r := range m.Replacements {
				item.Suggestions = append(item.Suggestions, r.Value)
			}
		}

		out.Items = append(out.Items, item)
	}

	return out, nil
}
