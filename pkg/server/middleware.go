package server

import (
	"net/http"
	"runtime/debug"

	"github.com/thedadams/clicky-serves/pkg/context"
	"github.com/thedadams/clicky-serves/pkg/log"
)

type middleware func(http.Handler) http.Handler

func apply(h http.Handler, m ...func(http.Handler) http.Handler) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}

func contentType(contentTypes ...string) middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, ct := range contentTypes {
				w.Header().Add("Content-Type", ct)
			}
			h.ServeHTTP(w, r)
		})
	}
}

func logRequest(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := context.GetLogger(r.Context())

		defer func() {
			if err := recover(); err != nil {
				l.Error("Panic", "error", err, "stack", string(debug.Stack()))
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error": "encountered an unexpected error"}`))
			}
		}()

		l.Info("Handling request", "method", r.Method, "path", r.URL.Path)
		h.ServeHTTP(w, r)
		l.Info("Handled request", "method", r.Method, "path", r.URL.Path)
	})
}

func addRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithNewRequestID(r.Context())))
	})
}

func addLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(
			w,
			r.WithContext(context.WithLogger(
				r.Context(),
				log.NewWithID(context.GetRequestID(r.Context())),
			)),
		)
	})
}
