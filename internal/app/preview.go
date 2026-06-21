package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const (
	previewMaxChars = 140
	scannerMaxSize  = 10 * 1024 * 1024
	maxCwdLines     = 200
	maxPreviewLines = 400
	modelReadChunk  = 64 * 1024
)

var claudeModelRe = regexp.MustCompile(`^claude-(fable|opus|sonnet|haiku)-(\d+)(?:-(\d+))?`)

func ExtractPreview(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerMaxSize)

	var summary string
	var userText string
	var assistantText string
	lines := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		rec, ok := parseRecord(line)
		if !ok {
			continue
		}

		if rec.Type == "summary" && rec.Summary != "" {
			summary = rec.Summary
			break
		}
		if userText == "" && isUserRecord(rec) {
			if content := extractContentText(rec.MessageContent); content != "" {
				userText = content
			}
		}
		if assistantText == "" && isAssistantRecord(rec) {
			if content := extractContentText(rec.MessageContent); content != "" {
				assistantText = content
			}
		}

		lines++
		if lines >= maxPreviewLines && summary == "" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		if summary != "" {
			return normalizePreview(summary), nil
		}
		if userText != "" {
			return normalizePreview(userText), nil
		}
		if assistantText != "" {
			return normalizePreview(assistantText), nil
		}
		return "", err
	}

	if summary != "" {
		return normalizePreview(summary), nil
	}
	if userText != "" {
		return normalizePreview(userText), nil
	}
	if assistantText != "" {
		return normalizePreview(assistantText), nil
	}

	return "", nil
}

func ExtractFirstUserText(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerMaxSize)

	lines := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		rec, ok := parseRecord(line)
		if !ok {
			continue
		}

		if isUserRecord(rec) {
			if content := extractContentText(rec.MessageContent); content != "" {
				return normalizeWhitespace(content), nil
			}
		}

		lines++
		if lines >= maxPreviewLines {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}

type record struct {
	Type           string          `json:"type"`
	Summary        string          `json:"summary"`
	Message        message         `json:"message"`
	MessageContent json.RawMessage `json:"-"`
}

type message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	Model   string          `json:"model"`
}

func parseRecord(line string) (record, bool) {
	var rec record
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		return record{}, false
	}
	if len(rec.Message.Content) > 0 {
		rec.MessageContent = rec.Message.Content
	}
	return rec, true
}

func isUserRecord(rec record) bool {
	if rec.Type == "user" {
		return true
	}
	if rec.Message.Role == "user" {
		return true
	}
	return false
}

func isAssistantRecord(rec record) bool {
	if rec.Type == "assistant" {
		return true
	}
	if rec.Message.Role == "assistant" {
		return true
	}
	return false
}

func extractContentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var str string
	if err := json.Unmarshal(raw, &str); err == nil && str != "" {
		return str
	}

	var items []interface{}
	if err := json.Unmarshal(raw, &items); err == nil {
		for _, item := range items {
			if text := extractTextItem(item); text != "" {
				return text
			}
		}
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err == nil {
		if text := extractTextItem(obj); text != "" {
			return text
		}
	}

	return ""
}

func extractTextItem(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]interface{}:
		if t, ok := v["type"].(string); ok && t != "" && t != "text" {
			return ""
		}
		if text, ok := v["text"].(string); ok && text != "" {
			return text
		}
	}

	return ""
}

func normalizePreview(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	text = normalizeWhitespace(text)
	return truncate(text, previewMaxChars)
}

func normalizeWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func truncate(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	if max <= 3 {
		return string(runes[:max])
	}

	return string(runes[:max-3]) + "..."
}

func ExtractCWD(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerMaxSize)

	lines := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var rec struct {
			CWD string `json:"cwd"`
		}
		if err := json.Unmarshal([]byte(line), &rec); err == nil && rec.CWD != "" {
			return rec.CWD, nil
		}

		lines++
		if lines >= maxCwdLines {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}

func ExtractSessionModel(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}

	var carry []byte
	for offset := info.Size(); offset > 0; {
		size := int64(modelReadChunk)
		if offset < size {
			size = offset
		}
		offset -= size

		chunk := make([]byte, size)
		if _, err := file.ReadAt(chunk, offset); err != nil && err != io.EOF {
			return "", err
		}
		chunk = append(chunk, carry...)
		lines := strings.Split(string(chunk), "\n")

		start := 0
		if offset > 0 {
			carry = []byte(lines[0])
			start = 1
		} else {
			carry = nil
		}

		for i := len(lines) - 1; i >= start; i-- {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}
			rec, ok := parseRecord(line)
			if !ok || !isAssistantRecord(rec) {
				continue
			}
			model := strings.TrimSpace(rec.Message.Model)
			if model != "" && model != "<synthetic>" {
				return model, nil
			}
		}
	}

	return "", nil
}

func ModelDisplayName(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "Session model unknown"
	}
	if matches := claudeModelRe.FindStringSubmatch(model); len(matches) == 4 {
		family := strings.ToUpper(matches[1][:1]) + matches[1][1:]
		if matches[3] == "" {
			return fmt.Sprintf("Claude %s %s", family, matches[2])
		}
		return fmt.Sprintf("Claude %s %s.%s", family, matches[2], matches[3])
	}
	return model
}
