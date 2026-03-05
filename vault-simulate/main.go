package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand/v2" // Using math/rand/v2 for non-cryptographic random intervals
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/vault/api"
)

func main() {
	// CLI flags
	duration := flag.Duration("duration", 60*time.Second, "Duration to run the simulation (e.g., 30s, 5m, 1h)")
	numRequests := flag.Int("num-requests", 100, "Number of requests to make during the duration")
	minInterval := flag.Duration("min-interval", 100*time.Millisecond, "Minimum interval between requests (e.g., 100ms, 1s)")
	maxInterval := flag.Duration("max-interval", 3*time.Second, "Maximum interval between requests (e.g., 1s, 10s)")

	flag.Parse()

	// Validate inputs
	if *numRequests <= 0 {
		log.Fatal("num-requests must be greater than 0")
	}
	if *duration <= 0 {
		log.Fatal("duration must be greater than 0")
	}
	if *minInterval <= 0 {
		log.Fatal("min-interval must be greater than 0")
	}
	if *maxInterval <= *minInterval {
		log.Fatal("max-interval must be greater than min-interval")
	}

	// Create Vault client configuration using SDK defaults
	// The SDK automatically reads VAULT_ADDR, VAULT_TOKEN and other standard env vars
	config := api.DefaultConfig()

	client, err := api.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create Vault client: %v", err)
	}

	// Validate that required environment variables are set
	if client.Token() == "" {
		log.Fatal("VAULT_TOKEN environment variable is required")
	}

	// Setup context with cancellation for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received interrupt signal, shutting down...")
		cancel()
	}()

	log.Printf("Starting vault-simulate: duration=%s, num-requests=%d, vault-addr=%s (from VAULT_ADDR)",
		*duration, *numRequests, config.Address)

	// Run the simulation
	if err := simulate(ctx, client, *duration, *numRequests, *minInterval, *maxInterval); err != nil {
		log.Fatalf("Simulation failed: %v", err)
	}

	log.Println("Simulation completed successfully")
}

// simulate runs the traffic simulation against Vault
func simulate(ctx context.Context, client *api.Client, duration time.Duration, numRequests int, minInterval, maxInterval time.Duration) error {
	intervals := generateRandomIntervals(duration, numRequests, minInterval, maxInterval)

	requestCount := 0
	successCount := 0
	errorCount := 0

	for i, interval := range intervals {
		select {
		case <-ctx.Done():
			log.Printf("Simulation stopped: completed %d/%d requests (success: %d, errors: %d)",
				requestCount, numRequests, successCount, errorCount)
			return nil
		case <-time.After(interval):
			requestCount++
			if err := makeVaultRequest(client, i); err != nil {
				errorCount++
				log.Printf("Request %d failed: %v", requestCount, err)
			} else {
				successCount++
				log.Printf("Request %d completed successfully", requestCount)
			}
		}
	}

	log.Printf("Simulation finished: completed %d/%d requests (success: %d, errors: %d)",
		requestCount, numRequests, successCount, errorCount)
	return nil
}

// generateRandomIntervals creates random intervals between minInterval and maxInterval
// that are distributed across the total duration
func generateRandomIntervals(totalDuration time.Duration, numRequests int, minInterval, maxInterval time.Duration) []time.Duration {
	intervals := make([]time.Duration, numRequests)

	// Calculate average interval needed
	avgInterval := totalDuration / time.Duration(numRequests)

	// Generate random intervals within bounds
	for i := range numRequests {
		// Generate a random interval between min and max
		intervalRange := maxInterval - minInterval
		randomOffset := time.Duration(rand.Int64N(int64(intervalRange)))
		interval := minInterval + randomOffset

		// Scale to fit within the total duration constraints
		if avgInterval < minInterval {
			interval = minInterval
		} else if avgInterval > maxInterval {
			interval = maxInterval
		} else {
			// Use a weighted random between the average and our random interval
			interval = (interval + avgInterval) / 2
		}

		intervals[i] = interval
	}

	return intervals
}

// makeVaultRequest performs a simulated request to Vault
func makeVaultRequest(client *api.Client, requestNum int) error {
	// Generate a unique key for this request
	key := fmt.Sprintf("simulate/key-%d-%d", requestNum, time.Now().UnixNano())
	secretData := map[string]interface{}{
		"value":     fmt.Sprintf("simulated-value-%d", requestNum),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Write a secret to KV v2
	_, err := client.KVv2("secret").Put(context.Background(), key, secretData)
	if err != nil {
		return fmt.Errorf("failed to write secret: %w", err)
	}

	return nil
}
