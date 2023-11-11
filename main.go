package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/josecleiton/domino/app/controllers"
)

func main() {
	http.HandleFunc("/", controllers.PlayHandler)

	port := ":8080"

	fmt.Printf("Server started at http://localhost%s\n", port)

	err := http.ListenAndServe(port, nil)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

}
