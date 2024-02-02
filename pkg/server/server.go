package main

import (
	"log"
	"net/http"

	"github.com/CloudyKit/jet/v6"
)

// TODO: Replace with service and load static files.
var views = jet.NewSet(jet.NewOSFileSystemLoader("templates"))

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		view, err := views.GetTemplate("/partials/test.jet")
		if err != nil {
			log.Println("Unexpected template error:", err.Error())
		}
		view.Execute(w, nil, nil)
	})

	http.ListenAndServe(":8080", nil)
}
