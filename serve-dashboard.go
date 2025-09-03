package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// Get current directory
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting current directory:", err)
	}

	// Create a file server to serve static files
	fs := http.FileServer(http.Dir(dir))
	
	// Handle root path to serve dashboard.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(dir, "dashboard.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})

	// Enable CORS for local development
	http.HandleFunc("/contract_addresses.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		http.ServeFile(w, r, filepath.Join(dir, "contract_addresses.json"))
	})

	port := "3000"
	fmt.Printf("🌐 PoCW Dashboard Server starting...\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📊 Dashboard: http://localhost:%s\n", port)
	fmt.Printf("📁 Serving files from: %s\n", dir)
	fmt.Printf("🔗 Make sure Anvil is running on port 8545\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}