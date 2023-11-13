package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/josecleiton/domino/app/controllers"
)

func main() {
	http.HandleFunc("/", controllers.GameHandler)

	port := ":8080"

	log.Printf("Server started at http://localhost%s\n", port)

	err := http.ListenAndServe(port, nil)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

}
