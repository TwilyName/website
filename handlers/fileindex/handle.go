package fileindex

import (
	"fmt"
	"net/http"

	"github.com/TwilyName/website/config"
	"github.com/TwilyName/website/handlers/errors"
	"github.com/TwilyName/website/handlers/markdown"
	"github.com/TwilyName/website/handlers/util"
	"github.com/TwilyName/website/log"
	tpl "github.com/TwilyName/website/template"
)

type uploader func(w http.ResponseWriter, r *http.Request, params searchParams) (bool, error)

type handler struct {
	root      http.FileSystem
	path      string
	endpoint  config.FileindexHandlerEndpointStruct
	uploaders map[string]uploader
}

func CreateHandler(
	root http.FileSystem,
	path string,
	endpoint config.FileindexHandlerEndpointStruct,
) http.Handler {
	tpl.AssertExists("fileindex")

	h := &handler{root, path, endpoint, map[string]uploader{}}
	h.uploaders = map[string]uploader{
		"tar": h.uploadTar,
		"gz":  h.uploadGz,
		"zst": h.uploadZst,
	}

	return h
}

type preservedParam struct {
	Key   string
	Value string
}

type page struct {
	util.BreadcrumbData
	markdown.InlineMarkdown
	searchParams
	sortParams

	PreservedParams []preservedParam
	URL             string
	AllowUpload     bool
	List            []fileEntry
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remoteAddr := util.GetRemoteAddr(r)

	allowUpload := config.IsAllowedByACL(remoteAddr, h.endpoint.Upload)
	allowPost := r.Method == http.MethodPost && allowUpload
	allowView := r.Method != http.MethodPost && config.IsAllowedByACL(remoteAddr, h.endpoint.View)
	if !allowPost && !allowView {
		errors.WriteNotFoundError(w, r)
		return
	}

	if !errors.AssertPathBeginning(h.path, w, r) {
		return
	}

	query := r.URL.Query()

	sort := query.Get("sort")
	sortDesc := query.Has("sort") && sort[len(sort)-4:] == "desc"
	sortBySize := query.Has("sort") && sort[0:4] == "size"
	sortByDate := query.Has("sort") && sort[0:4] == "date"
	sortByName := !query.Has("sort") || (!sortBySize && !sortByDate)
	if sortByName {
		sort = string(SortByName)
	} else if sortBySize {
		sort = string(SortBySize)
	} else if sortByDate {
		sort = string(SortByDate)
	}

	pageData := page{
		BreadcrumbData:  util.PrepareBreadcrumb(r),
		AllowUpload:     allowUpload,
		URL:             r.URL.Path,
		PreservedParams: h.preserveGetParams(r),
		searchParams: searchParams{
			FindQuery:     query.Get("query"),
			FindMatchCase: query.Get("matchcase") == "on",
			FindRegex:     query.Get("regex") == "on",
		},
		sortParams: sortParams{
			IsDesc: sortDesc,
			Field:  SortBy(sort),
		},
	}
	hasQuery := len(pageData.searchParams.FindQuery) > 0
	pageData.AllowUpload = pageData.AllowUpload && !hasQuery

	if !config.Authenticate(r, h.endpoint.Auth) {
		authHeader := fmt.Sprintf(`Basic realm="Authentication required to use %s"`, pageData.LastBreadcrumb)
		w.Header().Set("WWW-Authenticate", authHeader)
		errors.WriteUnauthorizedError(w, r)
		return
	}

	if recv, err := h.recvFile(w, r); recv {
		return
	} else if err != nil {
		errors.WriteError(w, r, err)
		return
	}

	if sent, err := h.sendFile(w, r, pageData.searchParams); sent {
		return
	} else if err != nil {
		errors.WriteNotFoundError(w, r)
		return
	}

	if list, err := h.prepareFileList(r.URL.Path, remoteAddr, pageData.searchParams, pageData.sortParams); err != nil {
		errors.WriteError(w, r, err)
		return
	} else {
		pageData.List = list

		show, name := h.showMarkdown(list)
		ptype := config.PreviewNone
		if show && !hasQuery {
			ptype = h.endpoint.Preview
			path := fmt.Sprintf("%s/%s", r.URL.Path, name)
			file, _ := h.root.Open(path)
			pageData.InlineMarkdown = markdown.PrepareInline(ptype, file)
			file.Close()
		}
	}

	err := util.MinifyTemplate("fileindex", pageData, w)
	if err != nil {
		log.Stderr().Print(err)
	}
}

func (h *handler) preserveGetParams(r *http.Request) []preservedParam {
	result := make([]preservedParam, 0)
	for key, value := range r.URL.Query() {
		if len(value) > 0 {
			result = append(result, preservedParam{Key: key, Value: value[0]})
		}
	}
	return result
}
