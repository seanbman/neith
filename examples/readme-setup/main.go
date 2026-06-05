package main

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/snburman/fcmp"
)

const (
	updatesCacheKey = "admin_updates"
	maxRows         = 10
)

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

	return fcmp.NewFn(ctx, dashboard(ctx, "Fill out the form to add a cache update.")).
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
			return fcmp.NewFn(ctx, dashboard(ctx, fmt.Sprintf("Cache record #%03d was already gone.", id))).
				WithEvents(handleSubmit, fcmp.OnSubmit)
		}
		return fcmp.NewFn(ctx, dashboard(ctx, fmt.Sprintf("Deleted cache record #%03d.", id))).
			WithEvents(handleSubmit, fcmp.OnSubmit)
	default:
		if err := addUpdate(ctx, form); err != nil {
			return fcmp.FnErr(ctx, err)
		}
		return fcmp.NewFn(ctx, dashboard(ctx, "Cache updated from submitted form data.")).
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
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}
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

func dashboard(ctx context.Context, notice string) fcmp.HTML {
	updates, err := fcmp.UseCache[[]adminUpdate](ctx, updatesCacheKey)
	if err != nil {
		return fcmp.HTML(fmt.Sprintf(`<section><h1>Cache error</h1><p>%s</p></section>`, html.EscapeString(err.Error())))
	}

	history, _ := updates.History()
	rows := updates.Value()

	var b strings.Builder
	b.WriteString(`<section class="dashboard">`)
	b.WriteString(`<header class="dashboard__header">`)
	b.WriteString(`<div>`)
	b.WriteString(`<p class="eyebrow">Admin cache monitor</p>`)
	b.WriteString(`<h1>Manual update table</h1>`)
	b.WriteString(`<p class="muted">Submit the form to add real example data to fcmp cache and re-render this table.</p>`)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="status">`)
	b.WriteString(`<span class="status__dot"></span>`)
	b.WriteString(html.EscapeString(notice))
	b.WriteString(`</div>`)
	b.WriteString(`</header>`)

	b.WriteString(`<div class="metrics">`)
	b.WriteString(metric("Cached rows", len(rows)))
	b.WriteString(metric("History snapshots", len(history)))
	b.WriteString(metric("Max retained rows", maxRows))
	b.WriteString(`</div>`)

	b.WriteString(updateForm())
	b.WriteString(updateTable(rows))
	b.WriteString(recordWindow(rows))
	b.WriteString(historyWindow(history))
	b.WriteString(`</section>`)

	return fcmp.HTML(b.String())
}

func updateForm() string {
	return `<form class="update-form">
		<input type="hidden" name="intent" value="add">
		<label>
			<span>Source</span>
			<input name="source" value="Billing service">
		</label>
		<label>
			<span>Status</span>
			<select name="status">
				<option value="ok">ok</option>
				<option value="queued">queued</option>
				<option value="warning">warning</option>
			</select>
		</label>
		<label class="message-field">
			<span>Message</span>
			<input name="message" value="Invoice reconciliation completed">
		</label>
		<button type="submit">Add update</button>
	</form>`
}

func updateTable(rows []adminUpdate) string {
	var b strings.Builder
	b.WriteString(`<div class="table-wrap">`)
	b.WriteString(`<table>`)
	b.WriteString(`<thead><tr><th>ID</th><th>Received</th><th>Source</th><th>Status</th><th>Message</th><th>Actions</th></tr></thead>`)
	b.WriteString(`<tbody>`)
	if len(rows) == 0 {
		b.WriteString(`<tr><td colspan="6" class="empty">No cache updates yet. Submit the form above.</td></tr>`)
	}
	for _, row := range rows {
		b.WriteString(`<tr>`)
		b.WriteString(fmt.Sprintf(`<td>#%03d</td>`, row.ID))
		b.WriteString(fmt.Sprintf(`<td>%s</td>`, html.EscapeString(row.ReceivedAt.Format("15:04:05"))))
		b.WriteString(fmt.Sprintf(`<td>%s</td>`, html.EscapeString(row.Source)))
		b.WriteString(fmt.Sprintf(`<td><span class="pill pill--%s">%s</span></td>`, statusClass(row.Status), html.EscapeString(row.Status)))
		b.WriteString(fmt.Sprintf(`<td>%s</td>`, html.EscapeString(row.Message)))
		b.WriteString(fmt.Sprintf(`<td>
			<form class="delete-form">
				<input type="hidden" name="intent" value="delete">
				<input type="hidden" name="id" value="%d">
				<button class="button-danger" type="submit">Delete</button>
			</form>
		</td>`, row.ID))
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table>`)
	b.WriteString(`</div>`)
	return b.String()
}

func recordWindow(rows []adminUpdate) string {
	var b strings.Builder
	b.WriteString(`<section class="record-window">`)
	b.WriteString(`<div class="record-window__header">`)
	b.WriteString(`<h2>Full cache contents</h2>`)
	b.WriteString(`<p>Terminal-style dump of the current admin_updates cache value.</p>`)
	b.WriteString(`</div>`)

	if len(rows) == 0 {
		b.WriteString(`<pre class="cache-terminal"><code>admin_updates = []adminUpdate{}</code></pre>`)
		b.WriteString(`</section>`)
		return b.String()
	}

	b.WriteString(`<pre class="cache-terminal"><code>`)
	b.WriteString(html.EscapeString(formatCacheDump(rows)))
	b.WriteString(`</code></pre>`)
	b.WriteString(`</section>`)
	return b.String()
}

func historyWindow(history map[string][]adminUpdate) string {
	var b strings.Builder
	b.WriteString(`<section class="record-window">`)
	b.WriteString(`<div class="record-window__header">`)
	b.WriteString(`<h2>Cache history store</h2>`)
	b.WriteString(`<p>Every Set call records a version here because Record(true) is enabled.</p>`)
	b.WriteString(`</div>`)
	b.WriteString(`<pre class="cache-terminal cache-terminal--history"><code>`)
	b.WriteString(html.EscapeString(formatHistoryDump(history)))
	b.WriteString(`</code></pre>`)
	b.WriteString(`</section>`)
	return b.String()
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

func metric(label string, value int) string {
	return fmt.Sprintf(
		`<article class="metric"><strong>%d</strong><span>%s</span></article>`,
		value,
		html.EscapeString(label),
	)
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
