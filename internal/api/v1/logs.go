package v1

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/logging"
)

type logEntryBody struct {
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

type logListInput struct {
	Level string `query:"level" doc:"Filter by minimum level: debug, info, warn, error" enum:"debug,info,warn,error"`
	Limit int    `query:"limit" default:"100" minimum:"1" maximum:"1000"`
}

type logListOutput struct {
	Body []*logEntryBody
}

// RegisterLogRoutes registers the GET /api/v1/system/logs endpoint.
func RegisterLogRoutes(api huma.API, buf *logging.RingBuffer) {
	huma.Register(api, huma.Operation{
		OperationID: "list-logs",
		Method:      http.MethodGet,
		Path:        "/api/v1/system/logs",
		Summary:     "List recent log entries",
		Description: "Returns in-memory log entries from the ring buffer, oldest first.",
		Tags:        []string{"System"},
	}, func(_ context.Context, input *logListInput) (*logListOutput, error) {
		limit := input.Limit
		if limit == 0 {
			limit = 100
		}

		minLevel := parseLevelFilter(input.Level)
		entries := buf.Entries()

		// Filter by level and apply limit (take from the end = most recent).
		var filtered []*logEntryBody
		for i := len(entries) - 1; i >= 0 && len(filtered) < limit; i-- {
			e := entries[i]
			if levelValue(e.Level) < minLevel {
				continue
			}
			fields := e.Fields
			if len(fields) == 0 {
				fields = nil
			}
			filtered = append(filtered, &logEntryBody{
				Time:    e.Time,
				Level:   e.Level,
				Message: e.Message,
				Fields:  fields,
			})
		}

		// Reverse so oldest is first (chronological order).
		for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
			filtered[i], filtered[j] = filtered[j], filtered[i]
		}

		return &logListOutput{Body: filtered}, nil
	})
}

// parseLevelFilter returns the numeric threshold for a level filter string.
func parseLevelFilter(level string) int {
	switch strings.ToLower(level) {
	case "debug":
		return -4
	case "info":
		return 0
	case "warn":
		return 4
	case "error":
		return 8
	default:
		return -4 // no filter = show everything including debug
	}
}

// levelValue returns the numeric value for a level string as stored in entries.
func levelValue(level string) int {
	switch level {
	case "DEBUG":
		return -4
	case "INFO":
		return 0
	case "WARN":
		return 4
	case "ERROR":
		return 8
	default:
		return 0
	}
}
