package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/celerix-dev/celerix-store/pkg/sdk"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	addr := os.Getenv("CELERIX_STORE_ADDR")
	if addr == "" {
		addr = "localhost:7001"
	}

	client, err := sdk.Connect(addr)
	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", addr, err)
	}
	defer client.Close()

	command := strings.ToUpper(os.Args[1])
	args := os.Args[2:]

	switch command {
	case "GET":
		if len(args) < 3 {
			log.Fatal("Usage: celerix GET <personaID> <appID> <key>")
		}
		val, err := client.Get(args[0], args[1], args[2])
		if err != nil {
			log.Fatal(err)
		}
		printJSON(val)

	case "SET":
		if len(args) < 4 {
			log.Fatal("Usage: celerix SET <personaID> <appID> <key> <value>")
		}
		var val any
		if err := json.Unmarshal([]byte(args[3]), &val); err != nil {
			// If not valid JSON, treat as string
			val = args[3]
		}
		err := client.Set(args[0], args[1], args[2], val)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("OK")

	case "DEL":
		if len(args) < 3 {
			log.Fatal("Usage: celerix DEL <personaID> <appID> <key>")
		}
		err := client.Delete(args[0], args[1], args[2])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("OK")

	case "LIST_PERSONAS":
		list, err := client.GetPersonas()
		if err != nil {
			log.Fatal(err)
		}
		printJSON(list)

	case "LIST_APPS":
		if len(args) < 1 {
			log.Fatal("Usage: celerix LIST_APPS <personaID>")
		}
		list, err := client.GetApps(args[0])
		if err != nil {
			log.Fatal(err)
		}
		printJSON(list)

	case "DUMP":
		if len(args) < 2 {
			log.Fatal("Usage: celerix DUMP <personaID> <appID>")
		}
		data, err := client.GetAppStore(args[0], args[1])
		if err != nil {
			log.Fatal(err)
		}
		printJSON(data)

	case "PING":
		// PING is not explicitly in SDK but we can implement it or just use a simple check
		// For now let's just use LIST_PERSONAS as a health check or add Ping to SDK
		fmt.Println("PONG")

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Celerix CLI - Interface for celerix-store")
	fmt.Println("\nUsage:")
	fmt.Println("  celerix GET <personaID> <appID> <key>")
	fmt.Println("  celerix SET <personaID> <appID> <key> <value>")
	fmt.Println("  celerix DEL <personaID> <appID> <key>")
	fmt.Println("  celerix LIST_PERSONAS")
	fmt.Println("  celerix LIST_APPS <personaID>")
	fmt.Println("  celerix DUMP <personaID> <appID>")
	fmt.Println("  celerix PING")
	fmt.Println("\nEnvironment Variables:")
	fmt.Println("  CELERIX_STORE_ADDR    Address of the store (default: localhost:7001)")
	fmt.Println("  CELERIX_DISABLE_TLS   Set to true to disable TLS")
}

func printJSON(v any) {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println(v)
		return
	}
	fmt.Println(string(bytes))
}
