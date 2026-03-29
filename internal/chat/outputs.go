package chat

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"agent-os/internal/liveapp"
)

type parsedOutputBlock struct {
	Kind   string
	Format string
	Title  string
	Body   string
}

var arcFencePattern = regexp.MustCompile("(?s)```(arc-(?:diagram|demo|simulation|document))(?:\\s+([^\\n`]+))?\\n(.*?)\\n```")
var genericFencePattern = regexp.MustCompile("(?s)```([a-zA-Z0-9_-]+)\\n(.*?)\\n```")

func materializeAssistantOutputs(root string, session Session, turn int, prompt string, content string) (string, []Output, map[string]string) {
	cleaned, blocks := parseAssistantOutputBlocks(content)
	if len(blocks) == 0 {
		cleaned, blocks = parseFallbackOutputBlocks(prompt, content)
	}
	if len(blocks) == 0 {
		return cleaned, nil, nil
	}
	outputDir := filepath.Join(chatDir(root, session.ID), fmt.Sprintf("turn-%03d-outputs", turn))
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return cleaned, nil, nil
	}
	outputs := make([]Output, 0, len(blocks))
	artifacts := map[string]string{}
	for index, block := range blocks {
		output, artifactKey, ok := materializeOutputBlock(root, session, turn, index+1, prompt, outputDir, block)
		if !ok {
			continue
		}
		outputs = append(outputs, output)
		if output.Path != "" {
			artifacts[artifactKey] = output.Path
		}
	}
	if strings.TrimSpace(cleaned) == "" && len(outputs) > 0 {
		cleaned = defaultOutputSummary(outputs)
	}
	if len(outputs) == 0 {
		return cleaned, nil, nil
	}
	return cleaned, outputs, artifacts
}

func parseAssistantOutputBlocks(content string) (string, []parsedOutputBlock) {
	text := strings.ReplaceAll(content, "\r\n", "\n")
	matches := arcFencePattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return strings.TrimSpace(text), nil
	}
	blocks := make([]parsedOutputBlock, 0, len(matches))
	var cleaned strings.Builder
	cursor := 0
	for _, match := range matches {
		cleaned.WriteString(text[cursor:match[0]])
		cursor = match[1]

		kind := text[match[2]:match[3]]
		format := ""
		if match[4] >= 0 {
			format = strings.TrimSpace(text[match[4]:match[5]])
		}
		body := strings.TrimSpace(text[match[6]:match[7]])
		title := ""
		if lines := strings.Split(body, "\n"); len(lines) > 0 {
			first := strings.TrimSpace(lines[0])
			if strings.HasPrefix(strings.ToLower(first), "title:") {
				title = strings.TrimSpace(first[len("title:"):])
				body = strings.TrimSpace(strings.Join(lines[1:], "\n"))
			}
		}
		blocks = append(blocks, parsedOutputBlock{
			Kind:   strings.TrimPrefix(kind, "arc-"),
			Format: normalizeOutputFormat(kind, format),
			Title:  title,
			Body:   body,
		})
	}
	cleaned.WriteString(text[cursor:])
	return compactMarkdown(cleaned.String()), blocks
}

func parseFallbackOutputBlocks(prompt string, content string) (string, []parsedOutputBlock) {
	if !promptRequestsStructuredVisual(prompt) {
		return strings.TrimSpace(content), nil
	}
	text := strings.ReplaceAll(content, "\r\n", "\n")
	matches := genericFencePattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return strings.TrimSpace(text), nil
	}
	blocks := make([]parsedOutputBlock, 0, len(matches))
	var cleaned strings.Builder
	cursor := 0
	htmlKind := inferHTMLVisualKind(prompt)
	for _, match := range matches {
		lang := strings.ToLower(strings.TrimSpace(text[match[2]:match[3]]))
		body := strings.TrimSpace(text[match[4]:match[5]])
		block, ok := parseGenericFenceBlock(lang, body, htmlKind)
		if !ok {
			continue
		}
		cleaned.WriteString(text[cursor:match[0]])
		cursor = match[1]
		blocks = append(blocks, block)
	}
	if len(blocks) == 0 {
		return strings.TrimSpace(text), nil
	}
	cleaned.WriteString(text[cursor:])
	return compactMarkdown(cleaned.String()), blocks
}

func parseGenericFenceBlock(lang string, body string, htmlKind string) (parsedOutputBlock, bool) {
	switch lang {
	case "mermaid":
		return parsedOutputBlock{
			Kind:   "diagram",
			Format: "mermaid",
			Body:   body,
		}, true
	case "svg":
		if strings.Contains(strings.ToLower(body), "<svg") {
			return parsedOutputBlock{
				Kind:   "diagram",
				Format: "svg",
				Body:   body,
			}, true
		}
	case "html":
		return parsedOutputBlock{
			Kind:   htmlKind,
			Format: "html",
			Body:   body,
		}, true
	}
	return parsedOutputBlock{}, false
}

func normalizeOutputFormat(kind string, raw string) string {
	next := strings.ToLower(strings.TrimSpace(raw))
	if next != "" {
		return next
	}
	switch kind {
	case "arc-diagram":
		return "svg"
	case "arc-document":
		return "markdown"
	default:
		return "html"
	}
}

func compactMarkdown(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	out := make([]string, 0, len(lines))
	blank := false
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if strings.TrimSpace(trimmed) == "" {
			if blank {
				continue
			}
			blank = true
			out = append(out, "")
			continue
		}
		blank = false
		out = append(out, trimmed)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func materializeOutputBlock(root string, session Session, turn int, index int, prompt string, outputDir string, block parsedOutputBlock) (Output, string, bool) {
	baseID := fmt.Sprintf("turn-%03d-output-%02d", turn, index)
	title := strings.TrimSpace(block.Title)
	if title == "" {
		title = defaultOutputTitle(block.Kind, index)
	}
	output := Output{
		ID:         baseID,
		Kind:       block.Kind,
		Title:      title,
		Format:     block.Format,
		Inline:     block.Kind == "diagram" || block.Kind == "demo" || block.Kind == "simulation",
		Launchable: block.Kind == "demo" || block.Kind == "simulation",
		Status:     "ready",
	}

	filename := ""
	contents := block.Body
	switch block.Kind {
	case "diagram":
		filename = baseID + ".svg"
		if block.Format == "mermaid" {
			contents = renderMermaidToSVG(title, block.Body)
			output.Format = "svg"
		} else if !strings.Contains(contents, "<svg") {
			contents = renderFallbackDiagramSVG(title, block.Body)
			output.Format = "svg"
		}
		output.Preview = contents
	case "document":
		filename = baseID + ".md"
		output.Preview = contents
	case "demo", "simulation":
		filename = baseID + ".html"
		output.Preview = summarizeHTMLOutput(contents)
	default:
		return Output{}, "", false
	}

	fullPath := filepath.Join(outputDir, filename)
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		return Output{}, "", false
	}
	output.Path = fullPath

	if output.Launchable && shouldAutoLaunchOutput(prompt, session.Mode, output.Kind) {
		app, err := liveapp.StartStaticPreview(liveapp.StartOptions{
			Root:           root,
			SessionID:      session.ID,
			Title:          title,
			Origin:         baseID,
			Type:           output.Kind,
			SourcePath:     fullPath,
			AutoStopAfter:  20 * time.Minute,
			AutoStopPolicy: "idle_20m",
		})
		if err != nil {
			output.Status = "failed"
			output.Error = err.Error()
		} else {
			output.LiveAppID = app.ID
			output.PreviewURL = app.PreviewURL
		}
	}

	return output, block.Kind + "_" + fmt.Sprintf("%02d", index), true
}

func defaultOutputTitle(kind string, index int) string {
	switch kind {
	case "diagram":
		return "Схема"
	case "document":
		return "Документ"
	case "demo":
		return fmt.Sprintf("Демо %d", index)
	case "simulation":
		return fmt.Sprintf("Мини-симуляция %d", index)
	default:
		return fmt.Sprintf("Материал %d", index)
	}
}

func defaultOutputSummary(outputs []Output) string {
	parts := make([]string, 0, len(outputs))
	for _, output := range outputs {
		switch output.Kind {
		case "diagram":
			parts = append(parts, "схему")
		case "demo":
			parts = append(parts, "демо")
		case "simulation":
			parts = append(parts, "мини-симуляцию")
		case "document":
			parts = append(parts, "документ")
		}
	}
	if len(parts) == 0 {
		return "Готово."
	}
	return "Готово. Я добавил " + strings.Join(parts, ", ") + " ниже в разговоре."
}

func shouldAutoLaunchOutput(prompt string, mode string, kind string) bool {
	if kind != "demo" && kind != "simulation" {
		return false
	}
	if promptExplicitlyRequestsVisual(prompt) {
		return true
	}
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "study", "hero":
		return true
	default:
		return false
	}
}

func promptExplicitlyRequestsVisual(prompt string) bool {
	text := strings.ToLower(prompt)
	for _, marker := range []string{
		"мини-симуля", "симуляц", "simulation", "miniapp", "mini app", "мини приложение",
		"демо", "demo", "мини-сайт", "mini-site", "site", "interactive",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func promptExplicitlyRequestsDiagram(prompt string) bool {
	text := strings.ToLower(prompt)
	for _, marker := range []string{
		"схем", "диаграм", "diagram", "mermaid", "flowchart", "visualize", "визуализ",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func promptRequestsStructuredVisual(prompt string) bool {
	return promptExplicitlyRequestsVisual(prompt) || promptExplicitlyRequestsDiagram(prompt)
}

func inferHTMLVisualKind(prompt string) string {
	if promptExplicitlyRequestsVisual(prompt) {
		if strings.Contains(strings.ToLower(prompt), "симуля") || strings.Contains(strings.ToLower(prompt), "simulation") {
			return "simulation"
		}
	}
	return "demo"
}

func summarizeHTMLOutput(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "Интерактивный результат подготовлен."
	}
	if len(trimmed) > 200 {
		return "Интерактивный результат готов к запуску."
	}
	return "Интерактивный результат: " + trimmed
}

func renderFallbackDiagramSVG(title string, body string) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="920" height="320" viewBox="0 0 920 320" fill="none">
  <rect width="920" height="320" rx="28" fill="#F8FBFF"/>
  <rect x="18" y="18" width="884" height="284" rx="22" fill="white" stroke="#D7E3EF"/>
  <text x="46" y="74" fill="#13324B" font-family="Avenir Next,Segoe UI,sans-serif" font-size="28" font-weight="700">%s</text>
  <foreignObject x="44" y="100" width="832" height="174">
    <div xmlns="http://www.w3.org/1999/xhtml" style="font-family:Avenir Next,Segoe UI,sans-serif;font-size:18px;line-height:1.55;color:#4F667F;white-space:pre-wrap;">%s</div>
  </foreignObject>
</svg>`, html.EscapeString(title), html.EscapeString(strings.TrimSpace(body)))
}

func renderMermaidToSVG(title string, body string) string {
	nodes, layout := parseMermaidFlow(body)
	if len(nodes) == 0 {
		return renderFallbackDiagramSVG(title, body)
	}
	cardWidth := 220
	cardHeight := 88
	gap := 28
	width := 120 + len(nodes)*(cardWidth+gap)
	height := 260
	if layout == "vertical" {
		width = 720
		height = 140 + len(nodes)*(cardHeight+gap)
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" fill="none">`, width, height, width, height))
	b.WriteString(`<defs><filter id="shadow" x="-20%" y="-20%" width="140%" height="140%"><feDropShadow dx="0" dy="10" stdDeviation="10" flood-color="#1C2A3A" flood-opacity="0.12"/></filter></defs>`)
	b.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" rx="28" fill="#F6FAFF"/>`, width, height))
	b.WriteString(fmt.Sprintf(`<text x="44" y="54" fill="#13324B" font-family="Avenir Next,Segoe UI,sans-serif" font-size="28" font-weight="700">%s</text>`, html.EscapeString(title)))
	for index, node := range nodes {
		x := 44 + index*(cardWidth+gap)
		y := 96
		if layout == "vertical" {
			x = 44
			y = 84 + index*(cardHeight+gap)
		}
		b.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="20" fill="white" stroke="#D7E3EF" filter="url(#shadow)"/>`, x, y, cardWidth, cardHeight))
		b.WriteString(fmt.Sprintf(`<foreignObject x="%d" y="%d" width="%d" height="%d"><div xmlns="http://www.w3.org/1999/xhtml" style="display:flex;align-items:center;justify-content:center;width:100%%;height:100%%;padding:14px;text-align:center;font-family:Avenir Next,Segoe UI,sans-serif;font-size:16px;line-height:1.35;color:#29445F;">%s</div></foreignObject>`, x+10, y+8, cardWidth-20, cardHeight-16, html.EscapeString(node)))
		if index < len(nodes)-1 {
			if layout == "vertical" {
				ax := x + cardWidth/2
				ay := y + cardHeight + 8
				by := y + cardHeight + gap - 8
				b.WriteString(fmt.Sprintf(`<path d="M%d %d L%d %d" stroke="#F59B23" stroke-width="4" stroke-linecap="round"/>`, ax, ay, ax, by))
				b.WriteString(fmt.Sprintf(`<path d="M%d %d L%d %d L%d %d" fill="none" stroke="#F59B23" stroke-width="4" stroke-linecap="round" stroke-linejoin="round"/>`, ax-8, by-10, ax, by, ax+8, by-10))
			} else {
				ax := x + cardWidth + 8
				ay := y + cardHeight/2
				bx := x + cardWidth + gap - 8
				b.WriteString(fmt.Sprintf(`<path d="M%d %d L%d %d" stroke="#F59B23" stroke-width="4" stroke-linecap="round"/>`, ax, ay, bx, ay))
				b.WriteString(fmt.Sprintf(`<path d="M%d %d L%d %d L%d %d" fill="none" stroke="#F59B23" stroke-width="4" stroke-linecap="round" stroke-linejoin="round"/>`, bx-10, ay-8, bx, ay, bx-10, ay+8))
			}
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func parseMermaidFlow(body string) ([]string, string) {
	lines := strings.Split(body, "\n")
	layout := "horizontal"
	nodes := []string{}
	seen := map[string]struct{}{}
	labelPattern := regexp.MustCompile(`([A-Za-z0-9_]+)\[(.*?)\]`)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "graph td") || strings.HasPrefix(strings.ToLower(trimmed), "flowchart td") {
			layout = "vertical"
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "graph lr") || strings.HasPrefix(strings.ToLower(trimmed), "flowchart lr") {
			layout = "horizontal"
			continue
		}
		matches := labelPattern.FindAllStringSubmatch(trimmed, -1)
		for _, match := range matches {
			label := strings.TrimSpace(match[2])
			if label == "" {
				continue
			}
			if _, ok := seen[label]; ok {
				continue
			}
			seen[label] = struct{}{}
			nodes = append(nodes, label)
		}
	}
	return nodes, layout
}
