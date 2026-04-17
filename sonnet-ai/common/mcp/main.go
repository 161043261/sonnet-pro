package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	mcpclient "github.com/hangtiancheng/lark_ai_mcp/client"
	mcpserver "github.com/hangtiancheng/lark_ai_mcp/server"
)

func main() {
	// Define command line flags
	mode := flag.String("mode", "", "Run mode: server or client")
	httpAddr := flag.String("http-addr", ":8081", "HTTP server address")
	city := flag.String("city", "", "City name for weather query")
	flag.Parse()

	if *mode == "" {
		fmt.Println("Error: You must specify mode using --mode (server or client)")
		flag.Usage()
		os.Exit(1)
	}

	if *mode == "server" {
		// Start server
		fmt.Println("Starting MCP server...")
		if err := mcpserver.StartServer(*httpAddr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else if *mode == "client" {
		// Run client
		if *city == "" {
			fmt.Println("Error: You must specify city name using --city")
			flag.Usage()
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create client
		httpURL := "http://localhost:8081/mcp"
		mcpClient, err := mcpclient.NewMCPClient(httpURL)
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
		defer mcpClient.Close()

		// Initialize client
		if _, err := mcpClient.Initialize(ctx); err != nil {
			log.Fatalf("Initialization failed: %v", err)
		}

		// Execute health check
		if err := mcpClient.Ping(ctx); err != nil {
			log.Fatalf("Health check failed: %v", err)
		}

		// Call weather tool
		result, err := mcpClient.CallWeatherTool(ctx, *city)
		if err != nil {
			log.Fatalf("Tool call failed: %v", err)
		}

		// Display weather results
		fmt.Println("\nWeather query results:")
		fmt.Println(mcpClient.GetToolResultText(result))

		fmt.Println("\nClient initialized successfully. Shutting down...")
	}
}
