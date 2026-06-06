package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/snburman/fcmp"
)

const updatesCacheKey = "admin_updates"

type adminUpdate struct {
	ID         int
	Source     string
	Status     string
	Message    string
	ReceivedAt time.Time
}

func page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "static/index.html")
}

func app(ctx context.Context) fcmp.FnComponent {
	if err := ensureCache(ctx, updatesCacheKey, []adminUpdate{}); err != nil {
		return fcmp.FnErr(ctx, err)
	}

	updates, err := fcmp.UseCache[[]adminUpdate](ctx, updatesCacheKey)
	if err != nil {
		return fcmp.FnErr(ctx, err)
	}
	updates.Record(true)

	return fcmp.NewFn(ctx, dashboardView(ctx, "Fill out the form to add a cache update.")).
		WithEvents(handleSubmit, fcmp.OnSubmit)
}

func handleSubmit(ctx context.Context) fcmp.FnComponent {
	form, err := fcmp.EventData[map[string]string](ctx)
	if err != nil {
		return fcmp.FnErr(ctx, err)
	}

	switch form["intent"] {
	case "delete":
		id, err := strconv.Atoi(form["id"])
		if err != nil {
			return fcmp.FnErr(ctx, err)
		}
		deleted, err := deleteUpdate(ctx, id)
		if err != nil {
			return fcmp.FnErr(ctx, err)
		}
		if !deleted {
			return fcmp.NewFn(ctx, dashboardView(ctx, fmt.Sprintf("Cache record #%03d was already gone.", id))).
				WithEvents(handleSubmit, fcmp.OnSubmit)
		}
		return fcmp.NewFn(ctx, dashboardView(ctx, fmt.Sprintf("Deleted cache record #%03d.", id))).
			WithEvents(handleSubmit, fcmp.OnSubmit)
	default:
		if err := addUpdate(ctx, form); err != nil {
			return fcmp.FnErr(ctx, err)
		}
		return fcmp.NewFn(ctx, dashboardView(ctx, "Cache updated from submitted form data.")).
			WithEvents(handleSubmit, fcmp.OnSubmit)
	}
}

func addUpdate(ctx context.Context, form map[string]string) error {
	update := adminUpdate{
		Source:     clean(form["source"], "Manual entry"),
		Status:     clean(form["status"], "ok"),
		Message:    clean(form["message"], "No message provided"),
		ReceivedAt: time.Now(),
	}
	return appendUpdate(ctx, update)
}

func appendUpdate(ctx context.Context, update adminUpdate) error {
	updates, err := fcmp.UseCache[[]adminUpdate](ctx, updatesCacheKey)
	if err != nil {
		return err
	}

	rows := updates.Value()
	update.ID = nextID(rows)
	rows = append([]adminUpdate{update}, rows...)
	if err := updates.Set(rows); err != nil {
		return err
	}
	logCacheRecords(rows)
	return nil
}

func deleteUpdate(ctx context.Context, id int) (bool, error) {
	updates, err := fcmp.UseCache[[]adminUpdate](ctx, updatesCacheKey)
	if err != nil {
		return false, err
	}

	rows := updates.Value()
	next := make([]adminUpdate, 0, len(rows))
	deleted := false
	for _, row := range rows {
		if row.ID == id {
			deleted = true
			continue
		}
		next = append(next, row)
	}
	if !deleted {
		return false, nil
	}
	if err := updates.Set(next); err != nil {
		return false, err
	}
	logCacheRecords(next)
	return true, nil
}

func dashboardView(ctx context.Context, notice string) fcmp.Component {
	updates, err := fcmp.UseCache[[]adminUpdate](ctx, updatesCacheKey)
	if err != nil {
		return cacheError(err.Error())
	}

	history, _ := updates.History()
	return dashboard(notice, updates.Value(), history)
}

func formatCacheDump(rows []adminUpdate) string {
	var b strings.Builder
	b.WriteString("admin_updates = []adminUpdate{\n")
	b.WriteString(formatRows(rows, "  "))
	b.WriteString("}")
	return b.String()
}

func formatHistoryDump(history map[string][]adminUpdate) string {
	if len(history) == 0 {
		return "history = map[string][]adminUpdate{}"
	}

	keys := make([]string, 0, len(history))
	for key := range history {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("history = map[string][]adminUpdate{\n")
	for _, key := range keys {
		b.WriteString(fmt.Sprintf("  %q: []adminUpdate{\n", key))
		b.WriteString(formatRows(history[key], "    "))
		b.WriteString("  },\n\n")
	}
	b.WriteString("}")
	return b.String()
}

func formatRows(rows []adminUpdate, indent string) string {
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(indent + "{\n")
		b.WriteString(fmt.Sprintf("%s  ID:         %d,\n", indent, row.ID))
		b.WriteString(fmt.Sprintf("%s  Source:     %q,\n", indent, row.Source))
		b.WriteString(fmt.Sprintf("%s  Status:     %q,\n", indent, row.Status))
		b.WriteString(fmt.Sprintf("%s  Message:    %q,\n", indent, row.Message))
		b.WriteString(fmt.Sprintf("%s  ReceivedAt: %q,\n", indent, row.ReceivedAt.Format(time.RFC3339)))
		b.WriteString(indent + "},\n\n")
	}
	return b.String()
}

func nextID(rows []adminUpdate) int {
	next := 1
	for _, row := range rows {
		if row.ID >= next {
			next = row.ID + 1
		}
	}
	return next
}

func logCacheRecords(rows []adminUpdate) {
	fmt.Print(formatConsoleDump(rows))
}

func formatConsoleDump(rows []adminUpdate) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("============================================================\n")
	b.WriteString("admin_updates cache records\n")
	b.WriteString("============================================================\n")

	if len(rows) == 0 {
		b.WriteString("No records are currently stored in the cache.\n")
		b.WriteString("============================================================\n\n")
		return b.String()
	}

	for index, row := range rows {
		b.WriteString(fmt.Sprintf("Record %d of %d\n", index+1, len(rows)))
		b.WriteString(fmt.Sprintf("ID:          #%03d\n", row.ID))
		b.WriteString(fmt.Sprintf("Source:      %s\n", row.Source))
		b.WriteString(fmt.Sprintf("Status:      %s\n", row.Status))
		b.WriteString(fmt.Sprintf("Message:     %s\n", row.Message))
		b.WriteString(fmt.Sprintf("Received At: %s\n", row.ReceivedAt.Format(time.RFC3339)))
		b.WriteString("\n")
	}

	b.WriteString("============================================================\n\n")
	return b.String()
}

func ensureCache[T any](ctx context.Context, key string, initial T) error {
	_, err := fcmp.NewCache(ctx, key, initial)
	if err == nil || errors.Is(err, fcmp.ErrCacheExists) {
		return nil
	}
	return err
}

func clean(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func statusClass(status string) string {
	switch status {
	case "warning":
		return "warning"
	case "queued":
		return "queued"
	default:
		return "ok"
	}
}

func main() {
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("../../static/assets"))))
	http.HandleFunc("/", fcmp.MiddleWareFn(page, app))

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
