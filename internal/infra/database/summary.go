package database

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// OperationSummary reduces a SQL string to a safe "OPERATION table" summary for
// tracing. It returns only the verb and the first table name, so no bound value
// (an email, a password hash) can reach the span even in an interpolated
// statement.
func OperationSummary(sql string) string {
	fields := strings.Fields(strings.TrimSpace(sql))
	if len(fields) == 0 {
		return "unknown"
	}
	verb := strings.ToUpper(fields[0])
	table := targetTable(fields)
	if table == "" {
		return "unknown"
	}
	return verb + " " + table
}

// identifier trims identifier quotes from a token.
func identifier(s string) string { return strings.Trim(s, `"`) }

// targetTable returns the first table targeted by the statement. Only known
// write/select shapes are recognised; anything else yields "" so the caller can
// fall back to "unknown". Direct field access beats a regex over joined text,
// which loses word boundaries.
func targetTable(fields []string) string {
	if len(fields) < 2 {
		return ""
	}

	tail := fields[1:]
	switch strings.ToUpper(fields[0]) {
	case "SELECT":
		for i, f := range tail {
			if strings.EqualFold(f, "FROM") && i+1 < len(tail) {
				return identifier(tail[i+1])
			}
		}
	case "INSERT":
		if len(tail) > 1 && strings.EqualFold(tail[0], "INTO") {
			return identifier(tail[1])
		}
	case "DELETE":
		if len(tail) > 1 && strings.EqualFold(tail[0], "FROM") {
			return identifier(tail[1])
		}
	case "UPDATE", "TRUNCATE", "MERGE":
		return identifier(tail[0])
	}
	return ""
}

func tracerFromContext(ctx context.Context) trace.Span {
	span := trace.SpanFromContext(ctx)
	// A no-op span means no active tracing context; skip the work.
	if span == nil || !span.IsRecording() {
		return nil
	}
	return span
}
