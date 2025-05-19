package util

import (
	"io"

	tpl "github.com/TwilyName/website/template"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

var m = minify.New()

func InitializeMinify() {
	m.Add("text/html", &html.Minifier{
		KeepDocumentTags: true,
	})
}

func MinifyTemplate(name string, data interface{}, out io.Writer) error {
	bufr, bufw := io.Pipe()
	defer bufr.Close()

	go func() {
		err := tpl.Execute(name, data, bufw)
		if err != nil {
			bufw.CloseWithError(err)
		} else {
			bufw.Close()
		}
	}()

	if err := m.Minify("text/html", out, bufr); err != nil {
		return err
	}

	return nil
}
