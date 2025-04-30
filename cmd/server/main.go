package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/sudhanshuraheja/authllama/internal/auth"
)

func main() {
	store, err := auth.LoadAuthConfig("config/config.json")
	if err != nil {
		log.Fatalf("failed to load auth config: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		service := r.Header.Get("X-Service-Name")
		header := r.Header.Get("Authorization")

		if !store.IsAuthorized(service, header) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		fmt.Fprintf(w, "Hello, %s! You are authorized.\n", service)
	})

	port := "8080"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
