package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/celerix-dev/celerix-store/internal/api"
	"github.com/celerix-dev/celerix-store/internal/server"
	"github.com/celerix-dev/celerix-store/internal/vault"
	"github.com/celerix-dev/celerix-store/pkg/engine"
	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var frontendDist embed.FS

func main() {
	fmt.Println("Starting Celerix Store Daemon...")

	dataDir := os.Getenv("CELERIX_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	port := os.Getenv("CELERIX_PORT")
	if port == "" {
		port = "7001"
	}

	httpPort := os.Getenv("CELERIX_HTTP_PORT")
	if httpPort == "" {
		httpPort = "7002"
	}

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

	// 6. Initialize HTTP API & UI
	h := &api.Handler{Store: store}
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/personas", h.GetPersonas)
		apiGroup.GET("/personas/:persona/apps", h.GetApps)
		apiGroup.GET("/personas/:persona/apps/:app", h.GetAppStore)
		apiGroup.GET("/global/:app/:key", h.GetGlobal)
		apiGroup.POST("/personas/:persona/apps/:app/:key", h.Set)
		apiGroup.DELETE("/personas/:persona/apps/:app/:key", h.Delete)
		apiGroup.POST("/move", h.Move)
	}

	// Serve UI
	distFS, _ := fs.Sub(frontendDist, "dist")
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
			return
		}
		file, err := distFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			file.Close()
			http.FileServer(http.FS(distFS)).ServeHTTP(c.Writer, c.Request)
			return
		}
		c.FileFromFS("/", http.FS(distFS))
	})

	// 7. Start servers
	go func() {
		fmt.Printf("HTTP Management UI listening on :%s\n", httpPort)
		if err := r.Run(":" + httpPort); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// 8. Handle Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutdown signal received. Finalizing disk writes...")
		store.Wait()
		fmt.Println("Persistence complete. Exiting.")
		os.Exit(0)
	}()

	// 9. Start the TCP Server
	fmt.Printf("Celerix Engine listening on :%s (TCP)\n", port)
	err = router.Listen(port)
	if err != nil {
		select {
		case <-sigChan:
		default:
			log.Fatalf("TCP Server failed: %v", err)
		}
	}
}
