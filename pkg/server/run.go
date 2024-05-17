package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"

	"github.com/gptscript-ai/go-gptscript"
)

const callTypeConfirm = "callConfirm"

// execToolStreamWithEvents runs the tool with the given options, and streams the events to the response as server sent events.
func (s *server) execToolStreamWithEvents(ctx context.Context, l *slog.Logger, w http.ResponseWriter, opts gptscript.Opts, tool fmt.Stringer) {
	run, err := s.client.Evaluate(ctx, opts, tool)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to evaluate tool: %w", err))
		return
	}
	processEventStreamOutput(l, w, run)
}

// execFileStreamWithEvents runs the file with the given options, and streams the events to the response as server sent events.
func (s *server) execFileStreamWithEvents(ctx context.Context, l *slog.Logger, w http.ResponseWriter, opts gptscript.Opts, path string) {
	run, err := s.client.Run(ctx, path, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to run file: %w", err))
		return
	}
	processEventStreamOutput(l, w, run)
}

// processEventStreamOutput will stream the events of the tool to the response as server sent events.
// If an error occurs, then an event with the error will also be sent.
func processEventStreamOutput(l *slog.Logger, w http.ResponseWriter, run *gptscript.Run) {
	setStreamingHeaders(w)

	streamEvents(l, w, run.Events())

	// Read the output of the script.
	rawOut, err := run.RawOutput()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to read stdout: %w", err))
		return
	}

	stdErr := run.ErrorOutput()

	writeServerSentEvent(l, w, map[string]any{
		"stderr": run.ErrorOutput(),
	})
	writeServerSentEvent(l, w, map[string]any{
		"stdout": rawOut,
	})

	var execErrOutput string
	if errors.Is(err, context.DeadlineExceeded) {
		execErrOutput = "The tool call took too long to complete, aborting"
	} else if execErr := new(exec.ExitError); errors.As(err, &execErr) {
		execErrOutput = fmt.Sprintf("The tool call returned an exit code of %d with message %q and output %q", execErr.ExitCode(), execErr.String(), stdErr)
	} else if err != nil {
		execErrOutput = fmt.Sprintf("failed to wait: %v, error output: %s", err, stdErr)
	}

	if execErrOutput != "" {
		writeServerSentEvent(l, w, map[string]any{
			"stderr": execErrOutput,
		})
	}

	// Now that we have received all events, send the DONE event.
	_, err = w.Write([]byte("data: [DONE]\n\n"))
	if err == nil {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	l.Debug("wrote DONE event")
}

// streamEvents will stream the events of the tool to the response as server sent events.
// This looks for and tries to handle confirm events as well. However, that currently is not implemented in the SDK.
func streamEvents(l *slog.Logger, w http.ResponseWriter, events <-chan gptscript.Event) {
	var (
		lastRunID   string
		eventBuffer []gptscript.Event
	)

	l.Debug("receiving events")
	for e := range events {
		// Ensure that the callConfirm event is after an event with the same runID.
		if (len(eventBuffer) > 0 || e.Type == callTypeConfirm) && lastRunID != e.RunID {
			eventBuffer = append(eventBuffer, e)
			lastRunID = fmt.Sprint(e.RunID)
			continue
		}

		for _, ev := range eventBuffer {
			writeServerSentEvent(l, w, ev)
		}

		eventBuffer = nil
		lastRunID = fmt.Sprint(e.RunID)

		writeServerSentEvent(l, w, e)
	}

	l.Debug("done receiving events")
}

func writeResponse(w http.ResponseWriter, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to marshal response: %w", err))
		return
	}

	_, _ = w.Write(b)
}

func writeError(w http.ResponseWriter, code int, err error) {
	w.WriteHeader(code)
	resp := map[string]any{
		"stderr": err.Error(),
	}

	b, err := json.Marshal(resp)
	if err != nil {
		_, _ = w.Write([]byte(fmt.Sprintf(`{"stderr": "%s"}`, err.Error())))
		return
	}

	_, _ = w.Write(b)
}

func writeServerSentEvent(l *slog.Logger, w http.ResponseWriter, event any) {
	ev, err := json.Marshal(event)
	if err != nil {
		l.Warn("failed to marshal event", "error", err)
		return
	}

	_, err = w.Write([]byte(fmt.Sprintf("data: %s\n\n", ev)))
	if err == nil {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	l.Debug("wrote event", "event", string(ev))
}

func setStreamingHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}
