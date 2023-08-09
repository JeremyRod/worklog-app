package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

type myComponent struct {
	app.Compo
}

func (c *myComponent) Render() app.UI {
	return app.Div().Text("Hello, Go-App!")
}

func serveWebAssembly(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/wasm")
	http.ServeFile(w, r, "web/app.wasm")
}

func main() {
	app.Route("/", &myComponent{})
	app.RunWhenOnBrowser()

	http.HandleFunc("/wasm_exec.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "wasm_exec.js")
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("web/app.wasm", serveWebAssembly)

	fmt.Println("Server started at http://localhost:8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
