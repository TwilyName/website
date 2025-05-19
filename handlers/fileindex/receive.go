package fileindex

import (
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/TwilyName/website/config"
	"github.com/flytam/filenamify"
)

func (h *handler) recvFile(w http.ResponseWriter, r *http.Request) (bool, error) {
	if r.Method != http.MethodPost {
		return false, nil
	}

	r.ParseMultipartForm(1024 * 1024)

	file, header, err := r.FormFile("file")
	if err != nil {
		return false, errors.New("No file chosen")
	}
	defer file.Close()

	filename, err := filenamify.Filenamify(header.Filename, filenamify.Options{
		Replacement: "_",
	})
	if err != nil {
		return false, err
	}
	filepath := r.URL.Path + filename
	// TODO: debug! h.logger.Access.Printf("Receiving file '%s'", filepath)

	destFile, err := os.Create(config.Get().Paths.Base + filepath)
	if err != nil {
		return false, err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, file)
	if err != nil {
		return false, err
	}

	http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
	return true, nil
}
