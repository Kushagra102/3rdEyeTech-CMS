package main

import (
	"log"
	"net"
	"net/http"

	"github.com/zserge/lorca"
)

func main() {
	// Initialize Database
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Setup Router
	mux := http.NewServeMux()
	setupHandlers(mux, db)

	// Create a listener on a random available local port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}
	defer ln.Close()

	// Start Go Web Server
	go func() {
		log.Printf("Server starting gracefully on http://%s", ln.Addr().String())
		if err := http.Serve(ln, mux); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Launch Lorca Native UI pointing to the local Go server
	ui, err := lorca.New("http://"+ln.Addr().String(), "", 1400, 900, "--remote-allow-origins=*")
	if err != nil {
		log.Fatalf("Failed to start lorca browser UI: %v", err)
	}
	defer ui.Close()

	// Wait until the user closes the window UI
	<-ui.Done()
	log.Println("Shutting down Application...")
}
