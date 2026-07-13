// Entrypoint package for the LinkPulse application.
//
// @title           LinkPulse API
// @version         1.0
// @description     Production-grade URL shortener with analytics, authentication, and observability.
// @description
// @description     ## Authentication
// @description     All protected endpoints require a Bearer JWT in the Authorization header.
// @description     Obtain tokens via POST /api/v1/auth/login.
//
// @contact.name    JyotiBhardwajj
// @contact.url     https://github.com/JyotiBhardwajj/LinkPulse
//
// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT
//
// @host            localhost:8080
// @BasePath        /
//
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer " followed by your access token.
//
// @schemes http https
package main

import (
	"linkpulse/internal/app"
	"log"
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
