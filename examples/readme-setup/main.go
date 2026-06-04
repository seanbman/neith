package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/snburman/fcmp"
)

func page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "static/index.html")
}

func app(ctx context.Context) fcmp.FnComponent {
	if _, err := fcmp.NewCache(ctx, "count", 0); err != nil && err != fcmp.ErrCacheExists {
		return fcmp.FnErr(ctx, err)
	}
	return counter(ctx)
}

func counter(ctx context.Context) fcmp.FnComponent {
	count, err := fcmp.UseCache[int](ctx, "count")
	if err != nil {
		return fcmp.FnErr(ctx, err)
	}

	_ = count.Set(count.Value() + 1)

	return fcmp.NewFn(ctx, fcmp.HTML(fmt.Sprintf(`
		<section>
			<h1>fcmp external setup</h1>
			<p>This app imports fcmp like a consumer would.</p>
			<button>Clicked %d times</button>
		</section>
	`, count.Value()))).WithEvents(func(ctx context.Context) fcmp.FnComponent {
		return counter(ctx)
	}, fcmp.OnClick)
}

func main() {
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("../../static/assets"))))
	http.HandleFunc("/", fcmp.MiddleWareFn(page, app))

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
