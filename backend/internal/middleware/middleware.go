package middleware

import (
	"context"
	"net/http"
	"time"
)

type ctxKey int

const startTimeKey ctxKey = iota

// Timing stores the request start time in the context so handlers can report
// execution time (time_ns) in every response body.
func Timing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), startTimeKey, time.Now())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Elapsed returns the nanoseconds since the request started. If no start time is
// present (handler invoked outside the middleware), it returns 0.
func Elapsed(ctx context.Context) int64 {
	start, ok := ctx.Value(startTimeKey).(time.Time)
	if !ok {
		return 0
	}
	return time.Since(start).Nanoseconds()
}

// CORS allows the browser frontend (served from a different origin) to call the API.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
