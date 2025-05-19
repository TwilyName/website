package errors

import (
	"net/http"

	"github.com/TwilyName/website/handlers/util"
	"github.com/TwilyName/website/log"
	"github.com/pkg/errors"
)

type page struct {
	util.BreadcrumbData
	Error string
}

func WriteUnauthorizedError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	WriteError(w, r, errors.New("Unauthorized"))
}

func WriteNotFoundError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	WriteError(w, r, errors.New("Content not found"))
}

func WriteBadRequestError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	WriteError(w, r, errors.New("Bad request"))
}

func WriteError(w http.ResponseWriter, r *http.Request, message error) {
	tpl := page{
		BreadcrumbData: util.PrepareBreadcrumb(r),
		Error:          message.Error(),
	}

	err := util.MinifyTemplate("error", tpl, w)
	if err != nil {
		log.Stderr().Print(err)
	}
}
