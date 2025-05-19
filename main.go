package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/felixge/httpsnoop"
	getopt "github.com/pborman/getopt/v2"

	"github.com/TwilyName/website/config"
	"github.com/TwilyName/website/handlers"
	"github.com/TwilyName/website/handlers/errors"
	"github.com/TwilyName/website/handlers/fileindex"
	"github.com/TwilyName/website/handlers/markdown"
	"github.com/TwilyName/website/handlers/util"
	"github.com/TwilyName/website/log"
	"github.com/TwilyName/website/template"
)

var (
	configPath = "config.yaml"
	showHelp   = false
)

//go:embed static
var staticContent embed.FS

func main() {
	initialize()
	if showHelp {
		getopt.PrintUsage(os.Stdout)
		os.Exit(0)
	}

	config.Initialize(configPath)
	config := config.Get()

	log.Initialize()
	log.InitializeSignalHandler()
	template.Initialize()
	util.InitializeMinify()

	baseDir := http.Dir(config.Paths.Base)
	counter := 0

	for entry, endpoint := range config.Handlers.FileIndex.Endpoints {
		path := handlerPath(entry)
		handler := fileindex.CreateHandler(baseDir, path, endpoint)
		http.HandleFunc(path, wrapHandler(handler))
		counter += 1
	}

	for entry, endpoint := range config.Handlers.Graphviz.Endpoints {
		path := handlerPath(entry)
		handler := handlers.GraphvizHandler(path, endpoint)
		http.HandleFunc(path, wrapHandler(handler))
		counter += 1
	}

	for entry, endpoint := range config.Handlers.Webhook.Endpoints {
		path := handlerPath(entry)
		handler := handlers.WebhookHandler(path, endpoint)
		http.HandleFunc(path, wrapHandler(handler))
		counter += 1
	}

	for entry, endpoint := range config.Handlers.Cards.Endpoints {
		path := handlerPath(entry)
		handler := handlers.CardsHandler(path, endpoint)
		http.HandleFunc(path, wrapHandler(handler))
		counter += 1
	}

	for entry, endpoint := range config.Handlers.Markdown.Endpoints {
		path := handlerPath(entry)
		path = path[:len(path)-1]
		handler := markdown.CreateHandler(baseDir, path, endpoint)
		http.HandleFunc(path, wrapHandler(handler))
		counter += 1
	}

	log.Stdout().Printf("Total registered handlers: %v", counter)

	{
		http.Handle("/static/", http.FileServer(http.FS(staticContent)))
		log.Stdout().Print("Special 'static' handler registered")
	}

	var wg sync.WaitGroup
	for _, addr := range config.Listen {
		wg.Add(1)

		log.Stdout().Printf("Listening TCP on '%s'", addr)
		go func(addr string) {
			defer wg.Done()
			log.Stderr().Fatal(http.ListenAndServe(addr, nil))
		}(addr)
	}
	wg.Wait()
}

func handlerPath(name string) string {
	if name != "index" {
		return fmt.Sprintf("/%s/", name)
	}
	return "/"
}

func initialize() {
	getopt.FlagLong(&showHelp, "help", 'h', "Show usage information and exit.")
	getopt.FlagLong(&configPath, "config", 'c', "Path to configuration file.")

	getopt.Parse()
}

func wrapHandler(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !util.IsWhitelistedProxy(r) && config.Get().ReverseProxy.Policy == config.PolicyError {
			errors.WriteBadRequestError(w, r)
			return
		}

		if util.HandleThemeToggle(w, r) {
			return
		}

		metrics := httpsnoop.CaptureMetrics(handler, w, r)
		addr := util.GetRemoteAddr(r).String()
		domain := r.Host
		username, _, ok := r.BasicAuth()
		if !ok {
			username = "-"
		}
		timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 -0700 MST")
		request := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		referer := r.Header.Get("Referer")
		userAgent := r.Header.Get("User-Agent")
		logstring := fmt.Sprintf("%s %s %s [%s] \"%s\" %v %v %vms \"%s\" \"%s\"", addr, domain, username, timestamp, request, metrics.Code, metrics.Written, metrics.Duration.Milliseconds(), referer, userAgent)
		if metrics.Code < 400 {
			log.Access().Print(logstring)
		} else {
			log.Error().Print(logstring)
		}
	}
}
