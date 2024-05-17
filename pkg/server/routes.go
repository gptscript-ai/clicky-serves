package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	ccontext "github.com/gptscript-ai/clicky-serves/pkg/context"
	"github.com/gptscript-ai/go-gptscript"
)

const toolRunTimeout = 15 * time.Minute

type server struct {
	client *gptscript.Client
}

func addRoutes(mux *http.ServeMux, config Config) {
	s := &server{
		client: gptscript.NewClient(gptscript.ClientOpts{GPTScriptBin: config.GPTScriptBin}),
	}

	mux.HandleFunc("GET /healthz", s.health)

	mux.HandleFunc("GET /version", s.version)
	mux.HandleFunc("GET /list-tools", s.listTools)
	mux.HandleFunc("GET /list-models", s.listModels)

	mux.HandleFunc("POST /run", execFileHandler(s.execFileStreamWithEvents))
	mux.HandleFunc("POST /evaluate", execToolHandler(s.execToolStreamWithEvents))

	mux.HandleFunc("POST /parse", s.parse)
	mux.HandleFunc("POST /fmt", s.fmtDocument)
}

// health just provides an endpoint for checking whether the server is running and accessible.
func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	writeResponse(w, map[string]string{"status": "ok"})
}

// version will return the output of `gptscript --version`
func (s *server) version(w http.ResponseWriter, r *http.Request) {
	out, err := s.client.Version(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to get version: %w", err))
		return
	}

	writeResponse(w, map[string]any{"stdout": out})
}

// listTools will return the output of `gptscript --list-tools`
func (s *server) listTools(w http.ResponseWriter, r *http.Request) {
	out, err := s.client.ListTools(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to list tools: %w", err))
		return
	}

	writeResponse(w, map[string]any{"stdout": out})
}

// listModels will return the output of `gptscript --list-models`
func (s *server) listModels(w http.ResponseWriter, r *http.Request) {
	out, err := s.client.ListModels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to list models: %w", err))
		return
	}

	writeResponse(w, map[string]any{"stdout": strings.Join(out, "\n")})
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

		reqObject.IncludeEvents = true
		l.Debug("executing tool", "tool", reqObject)
		if reqObject.Content != "" {
			process(ctx, l, w, reqObject.Opts, &reqObject.content)
		} else {
			process(ctx, l, w, reqObject.Opts, &reqObject.ToolDef)
		}
	}
}

// execFileHandler is a general handler for executing files with gptscript. This is mainly responsible for parsing the request body.
// Then the options, path, and input are passed to the process function.
func execFileHandler(process func(ctx context.Context, l *slog.Logger, w http.ResponseWriter, opts gptscript.Opts, path string)) http.HandlerFunc {
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

		reqObject.IncludeEvents = true
		process(ctx, l, w, reqObject.Opts, reqObject.File)
	}
}

// parse will parse the file and return the corresponding Document.
func (s *server) parse(w http.ResponseWriter, r *http.Request) {
	reqObject := new(parseRequest)
	if err := json.NewDecoder(r.Body).Decode(reqObject); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	ctx := r.Context()
	l := ccontext.GetLogger(ctx)

	l.Debug("parsing file", "file", reqObject.File, "content", reqObject.Content)

	var (
		out []gptscript.Node
		err error
	)

	if reqObject.Content != "" {
		out, err = s.client.ParseTool(ctx, reqObject.Content)
	} else {
		out, err = s.client.Parse(ctx, reqObject.File)
	}
	if err != nil {
		l.Error("failed to parse file", "error", err)
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to parse file: %w", err))
		return
	}

	writeResponse(w, map[string]any{"stdout": map[string]any{"nodes": out}})
}

// fmtDocument will produce a string representation of the document.
func (s *server) fmtDocument(w http.ResponseWriter, r *http.Request) {
	doc := new(gptscript.Document)
	if err := json.NewDecoder(r.Body).Decode(doc); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	l := ccontext.GetLogger(r.Context())

	l.Debug("formatting document", "document", doc)

	ctx, cancel := context.WithTimeout(r.Context(), toolRunTimeout)
	defer cancel()

	out, err := s.client.Fmt(ctx, doc.Nodes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to format document: %w", err))
	}

	writeResponse(w, map[string]string{"stdout": out})
}
