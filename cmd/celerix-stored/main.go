package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/celerix-dev/celerix-store/internal/engine"
	"github.com/celerix-dev/celerix-store/internal/server"
	"github.com/celerix-dev/celerix-store/internal/vault"
)

func main() {
	fmt.Println("Starting Celerix Store Daemon...")

	// 1. Configuration (Could be moved to Env Vars later)
	dataDir := "./data"
	port := "7001"
	useTLS := os.Getenv("CELERIX_DISABLE_TLS") != "true"

	// 2. Initialize Persistence
	persister, err := engine.NewPersistence(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize persistence: %v", err)
	}

	// 3. Load existing data and start the Engine
	initialData, err := persister.LoadAll()
	if err != nil {
		log.Printf("Warning: Could not load existing data: %v", err)
	}

	store := engine.NewMemStore(initialData, persister)
	fmt.Printf("Engine started. Loaded %d personas.\n", len(initialData))

	// 4. Initialize the TCP Router
	router := server.NewRouter(store)

	// 5. Setup TLS
	if useTLS {
		fmt.Println("Generating self-signed certificate for internal TLS...")
		cert, err := vault.GenerateSelfSignedCert()
		if err != nil {
			log.Fatalf("Failed to generate TLS certificate: %v", err)
		}
		router.SetCertificate(cert)
		fmt.Println("TLS encryption enabled.")
	} else {
		fmt.Println("TLS encryption disabled (CELERIX_DISABLE_TLS=true).")
	}

	// 6. Handle Graceful Shutdown
	// This ensures that if the Docker container stops, we save everything.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan

		fmt.Println("\nShutdown signal received. Finalizing disk writes...")
		// Wait for background persistence tasks
		store.Wait()
		fmt.Println("Persistence complete. Exiting.")
		os.Exit(0)
	}()

	// 6. Start the Server
	fmt.Printf("Celerix Store listening on :%s\n", port)
	err = router.Listen(port)
	if err != nil {
		// Only fatal if it's not a normal shutdown
		select {
		case <-sigChan:
			// Already shutting down
		default:
			log.Fatalf("Server failed: %v", err)
		}
	}
}
