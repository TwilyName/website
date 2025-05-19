package handlers

import (
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/Twi1ightSpark1e/website/config"
	"github.com/Twi1ightSpark1e/website/handlers/errors"
	"github.com/Twi1ightSpark1e/website/handlers/util"
	"github.com/Twi1ightSpark1e/website/log"
	"github.com/google/shlex"
)

type webhookHandler struct {
	path     string
	endpoint config.WebhookEndpointStruct
}

func WebhookHandler(path string, endpoint config.WebhookEndpointStruct) http.Handler {
	return &webhookHandler{path, endpoint}
}

func (h *webhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remoteAddr := util.GetRemoteAddr(r)

	if !config.IsAllowedByACL(remoteAddr, h.endpoint.View) {
		errors.WriteNotFoundError(w, r)
		return
	}

	if !errors.AssertPath(h.path, w, r) {
		return
	}

	if h.endpoint.Method != "" && strings.EqualFold(r.Method, h.endpoint.Method) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for header, value := range h.endpoint.Headers {
		if r.Header[header][0] == value {
			continue
		}

		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)

	cmdline, _ := shlex.Split(h.endpoint.Exec)
	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	args := strings.Join(cmdline[1:], " ")
	if cmd == nil {
		log.Stderr().Printf("Cannot create process '%s' with arguments '%s'", cmdline[0], args)
		return
	}

	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Stderr().Printf("Cannot get stdin of spawned process: %v", err)
		return
	}

	err = cmd.Start()
	if err != nil {
		log.Stderr().Printf("Cannot spawn process: %v", err)
		return
	}
	// TODO: debug! h.logger.Access.Printf("Spawned process '%s' with arguments '%s'", cmdline[0], args)

	_, err = io.Copy(stdin, r.Body)
	if err != nil {
		log.Stderr().Printf("Cannot send request body to process stdin: %v", err)
		return
	}
	stdin.Close()

	err = cmd.Wait()
	if err != nil {
		log.Stderr().Printf("Cannot wait for webhook exit: %v", err)
	}

	cmd.ProcessState.ExitCode()
	// TODO: debug! h.logger.Access.Printf("Webhook exited with code %d", exitcode)
}
