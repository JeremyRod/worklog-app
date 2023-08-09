// main.go
package main

import (
	"fmt"
	"net/http"
)

func main() {
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	http.HandleFunc("/api/save", saveFormData)
	http.ListenAndServe(":8080", nil)
}

func saveFormData(w http.ResponseWriter, r *http.Request) {
	// Handle form data submission and store it
	fmt.Println("Form data received on the server")
}
