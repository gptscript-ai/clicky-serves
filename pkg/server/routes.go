package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gptscript-ai/go-gptscript"
	ccontext "github.com/thedadams/clicky-serves/pkg/context"
)

const toolRunTimeout = 15 * time.Minute

func addRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", health)

	mux.HandleFunc("GET /version", version)
	mux.HandleFunc("GET /list-tools", listTools)
	mux.HandleFunc("GET /list-models", listModels)

	mux.HandleFunc("POST /run-tool", execToolHandler(execTool))
	mux.HandleFunc("POST /run-tool-stream", execToolHandler(execToolStream))
	mux.HandleFunc("POST /run-tool-stream-with-events", execToolHandler(execToolStreamWithEvents))

	mux.HandleFunc("POST /run-file", execFileHandler(execFile))
	mux.HandleFunc("POST /run-file-stream", execFileHandler(execFileStream))
	mux.HandleFunc("POST /run-file-stream-with-events", execFileHandler(execFileStreamWithEvents))
}

// health just provides an endpoint for checking whether the server is running and accessible.
func health(w http.ResponseWriter, _ *http.Request) {
	writeResponse(w, map[string]string{"status": "ok"})
}

// version will return the output of `gptscript --version`
func version(w http.ResponseWriter, r *http.Request) {
	out, err := gptscript.Version(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to get version: %w", err))
		return
	}

	writeResponse(w, map[string]any{"output": out})
}

// listTools will return the output of `gptscript --list-tools`
func listTools(w http.ResponseWriter, r *http.Request) {
	out, err := gptscript.ListTools(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to list tools: %w", err))
		return
	}

	writeResponse(w, map[string]any{"output": out})
}

// listModels will return the output of `gptscript --list-models`
func listModels(w http.ResponseWriter, r *http.Request) {
	out, err := gptscript.ListModels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to list models: %w", err))
		return
	}

	writeResponse(w, map[string]any{"output": out})
}

// execToolHandler is a general handler for executing tools with gptscript. This is mainly responsible for parsing the request body.
// Then the options and tool are passed to the process function.
func execToolHandler(process func(ctx context.Context, l *slog.Logger, w http.ResponseWriter, opts gptscript.Opts, tool fmt.Stringer)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqObject := new(toolRequest)
		if err := json.NewDecoder(r.Body).Decode(reqObject); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), toolRunTimeout)
		defer cancel()

		l := ccontext.GetLogger(r.Context())

		l.Debug("executing tool", "tool", reqObject)
		if reqObject.Content != "" {
			process(ctx, l, w, reqObject.Opts, &reqObject.FreeForm)
		} else {
			process(ctx, l, w, reqObject.Opts, &reqObject.Tool)
		}
	}
}

// execFileHandler is a general handler for executing files with gptscript. This is mainly responsible for parsing the request body.
// Then the options, path, and input are passed to the process function.
func execFileHandler(process func(ctx context.Context, l *slog.Logger, w http.ResponseWriter, opts gptscript.Opts, path, input string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqObject := new(fileRequest)
		if err := json.NewDecoder(r.Body).Decode(reqObject); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
			return
		}

		l := ccontext.GetLogger(r.Context())

		l.Debug("executing file", "file", reqObject)

		ctx, cancel := context.WithTimeout(r.Context(), toolRunTimeout)
		defer cancel()

		process(ctx, l, w, reqObject.Opts, reqObject.File, reqObject.Input)
	}
}
