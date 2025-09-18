package template

import (
	"embed"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"

	"github.com/TwilyName/website/log"
	"github.com/shurcooL/httpfs/filter"
	"github.com/shurcooL/httpfs/vfsutil"
)

var suffix = ".tpl"
var templates *template.Template

//go:embed *.tpl base/*.tpl
var templatesContent embed.FS

func Initialize() {
	templates = template.New("")

	counter := 0
	root := filter.Keep(http.FS(templatesContent), func(path string, fi fs.FileInfo) bool {
		ok := fi.IsDir() || filepath.Ext(path) == suffix
		if !ok {
			log.Stderr().Printf("Invalid file embedded into binary: '%s'", fi.Name())
		}
		return ok
	})
	err := vfsutil.WalkFiles(root, ".", func(path string, fi fs.FileInfo, r io.ReadSeeker, err error) error {
		if err != nil {
			log.Stderr().Print(err)
			return nil
		}

		if path == "" || fi.IsDir() {
			return nil
		}

		content, err := io.ReadAll(r)
		if err != nil {
			log.Stderr().Print(err)
			return nil
		}

		path = path[:len(path)-len(suffix)]
		_, err = templates.New(path).Parse(string(content))
		if err != nil {
			log.Stderr().Print(err)
		} else {
			counter += 1
		}

		return nil
	})

	if err != nil {
		log.Stderr().Fatal(err)
	}

	log.Stdout().Printf("Total templates: %v", counter)
}

func Execute(name string, data interface{}, w io.Writer) error {
	return templates.ExecuteTemplate(w, name, data)
}

func AssertExists(name string) {
	for _, tpl := range templates.Templates() {
		if tpl.Name() == name {
			return
		}
	}

	log.Stderr().Fatalf("'%s' template is missing!", name)
}
