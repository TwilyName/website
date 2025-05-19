package fileindex

import (
	"html/template"
	"io"
	"strings"

	"github.com/TwilyName/website/handlers/markdown"
)

func useAsPreview(name string) bool {
	return strings.EqualFold(name, "readme.md")
}

func (h *handler) showMarkdown(list []fileEntry) (bool, string) {
	for _, file := range list {
		if file.IsDir || !useAsPreview(file.Name) {
			continue
		}
		return true, file.Name
	}
	return false, ""
}

func (h *handler) loadMarkdown(path string) (template.HTML, error) {
	file, err := h.root.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return markdown.Render(buf), nil
}
