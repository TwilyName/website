package markdown

import (
	"html/template"
	"io"
	"net/http"

	"github.com/TwilyName/website/config"
)

type InlineMarkdown struct {
	MarkdownVisibility config.PreviewType
	MarkdownTitle      string
	MarkdownContent    template.HTML
}

func PrepareInline(ptype config.PreviewType, file http.File) InlineMarkdown {
	res := InlineMarkdown{
		MarkdownVisibility: ptype,
	}

	stat, err := file.Stat()
	if err != nil {
		res.MarkdownVisibility = config.PreviewNone
		return res
	}
	res.MarkdownTitle = stat.Name()

	buf, err := io.ReadAll(file)
	if err != nil {
		res.MarkdownVisibility = config.PreviewNone
		return res
	}

	res.MarkdownContent = Render(buf)
	return res
}
