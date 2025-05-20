package handlers

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/TwilyName/website/config"
	handlerrors "github.com/TwilyName/website/handlers/errors"
	"github.com/TwilyName/website/handlers/util"
	"github.com/TwilyName/website/log"
	tpl "github.com/TwilyName/website/template"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
)

type graphvizPage struct {
	util.BreadcrumbData
	Image     template.HTML
	Timestamp string
}

type graphData struct {
	image     bytes.Buffer
	timestamp int64
}

type graphvizHandler struct {
	path     string
	endpoint config.GraphvizEndpointStruct
	graph    graphData
}

func GraphvizHandler(path string, endpoint config.GraphvizEndpointStruct) http.Handler {
	tpl.AssertExists("graphviz")
	return &graphvizHandler{path, endpoint, graphData{}}
}

func (h *graphvizHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tplData := graphvizPage{
		BreadcrumbData: util.PrepareBreadcrumb(r),
	}

	switch r.Method {
	case http.MethodPut:
		h.handlePUT(w, r)
		return
	case http.MethodDelete:
		h.handleDELETE(w, r)
		return
	case http.MethodGet:
		if !h.handleGET(w, r, &tplData) {
			return
		}
		err := util.MinifyTemplate("graphviz", tplData, w)
		if err != nil {
			log.Stderr().Print(err)
		}
	default:
		w.WriteHeader(http.StatusForbidden)
		handlerrors.WriteError(w, r, errors.New("Invalid request method"))
		return
	}
}

func (h *graphvizHandler) handlePUT(w http.ResponseWriter, r *http.Request) {
	remoteAddr := util.GetRemoteAddr(r)

	if !config.IsAllowedByACL(remoteAddr, h.endpoint.Edit) {
		handlerrors.WriteNotFoundError(w, r)
		return
	}

	if !handlerrors.AssertPath(h.path, w, r) {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		handlerrors.WriteError(w, r, err)
		return
	}

	ctx := context.Background()
	g, err := graphviz.New(ctx)
	if err != nil {
		handlerrors.WriteError(w, r, err)
		return
	}

	graph, err := graphviz.ParseBytes(body)
	if err != nil {
		handlerrors.WriteError(w, r, err)
		return
	}

	err = h.performGraphDecoration(g, graph)
	if err != nil {
		handlerrors.WriteError(w, r, err)
		return
	}

	// Render graph
	var buffer bytes.Buffer
	if err = g.Render(ctx, graph, "svg_inline", &buffer); err != nil {
		handlerrors.WriteError(w, r, err)
		return
	}

	h.performHtmlDecoration(&buffer)

	h.graph.image = buffer
	h.graph.timestamp = time.Now().Unix()

	w.Write([]byte("ok"))
}

func (h *graphvizHandler) handleDELETE(w http.ResponseWriter, r *http.Request) {
	remoteAddr := util.GetRemoteAddr(r)

	if !config.IsAllowedByACL(remoteAddr, h.endpoint.Edit) {
		handlerrors.WriteNotFoundError(w, r)
		return
	}

	if !handlerrors.AssertPath(h.path, w, r) {
		return
	}

	h.graph = graphData{}
	w.Write([]byte("ok"))
}

func (h *graphvizHandler) handleGET(w http.ResponseWriter, r *http.Request, tpl *graphvizPage) bool {
	remoteAddr := util.GetRemoteAddr(r)

	if !config.IsAllowedByACL(remoteAddr, h.endpoint.View) {
		handlerrors.WriteNotFoundError(w, r)
		return false
	}

	if !handlerrors.AssertPath(h.path, w, r) {
		return false
	}

	tpl.Image = template.HTML(h.graph.image.String())

	if h.graph.timestamp == 0 {
		tpl.Timestamp = "not performed yet"
	} else {
		tpl.Timestamp = time.Unix(h.graph.timestamp, 0).String()
	}

	return true
}

func (h *graphvizHandler) performGraphDecoration(g *graphviz.Graphviz, graph *cgraph.Graph) error {
	if h.endpoint.Decoration == config.DecorationTinc {
		g.SetLayout(graphviz.CIRCO)

		graph.SetBackgroundColor("transparent")

		var err error
		for node, err := graph.FirstNode(); node != nil && err == nil; node, err = graph.NextNode(node) {
			name, err := node.Name()
			if err != nil {
				continue
			}

			nodeLabel := strings.ReplaceAll(name, ".", "-")
			node.SetID(nodeLabel)

			if node.GetStr("style") == "filled" {
				node.SetFillColor(node.GetStr("color"))
			} else {
				node.SetStyle(cgraph.FilledNodeStyle).SetFillColor("#ffffff")
			}

			var neighbors []string
			for edge, err := graph.FirstEdge(node); edge != nil && err == nil; edge, err = graph.NextEdge(edge, node) {
				// TODO: this code will probably make it possible to highlight connected edges when hover on node sometime
				// class := edge.GetStr("class")
				// if len(class) == 0 {
				// 	edge.SafeSet("class", nodeLabel, "")
				// } else {
				// 	class = class + " " + nodeLabel
				// 	edge.Set("class", class)
				// }

				head, err := edge.Head()
				if err != nil {
					continue
				}
				headName, err := head.Name()
				if err != nil {
					continue
				}
				tail, err := edge.Tail()
				if err != nil {
					continue
				}
				tailName, err := tail.Name()
				if err != nil {
					continue
				}

				if name != headName {
					neighbors = append(neighbors, headName)
				} else {
					neighbors = append(neighbors, tailName)
				}
			}

			sort.Slice(neighbors, func(i, j int) bool {
				return strings.Compare(neighbors[i], neighbors[j]) < 0
			})
			tooltip := name + " is directly connected to:\n" + strings.Join(neighbors, ",\n")
			node.SetTooltip(tooltip)
		}

		return err
	}

	// Decoration is `none`, so nothing to do here
	return nil
}

func (h *graphvizHandler) performHtmlDecoration(buf *bytes.Buffer) error {
	svg := buf.String()

	{
		pattern := regexp.MustCompile(`(?sU)<svg.+width="(.+)".*>`)
		idxs := pattern.FindStringSubmatchIndex(svg)
		if len(idxs) > 0 {
			replaceStr := svg[idxs[2]:idxs[3]]
			svg = strings.Replace(svg, replaceStr, "85%", 1)
		}
	}
	{
		pattern := regexp.MustCompile(`(?s)<svg.+height="(.+?)".*?>`)
		idxs := pattern.FindStringSubmatchIndex(svg)
		if len(idxs) > 0 {
			replaceStr := svg[idxs[2]:idxs[3]]
			svg = strings.Replace(svg, replaceStr, "100%", 1)
		}
	}

	buf.Reset()
	_, err := buf.WriteString(svg)
	return err
}
