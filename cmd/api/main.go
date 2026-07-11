// Entrypoint package for the LinkPulse application.
package main

import (
	"log"
	"linkpulse/internal/app"
)

func main() {
	application, err := app.NewApplication()
	if err != nil {
		log.Fatalf("Failed to initialize application bootstrap: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Application execution encountered runtime error: %v", err)
	}
}
