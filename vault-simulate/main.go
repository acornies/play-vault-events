package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand/v2" // Using math/rand/v2 for non-cryptographic random intervals
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/vault/api"
)

// validRequestTypes defines the accepted request types for the simulation.
var validRequestTypes = map[string]bool{
	"kv":       true,
	"ldap":     true,
	"database": true,
}

// mountPaths maps request types to their expected Vault mount paths.
var mountPaths = map[string]string{
	"kv":       "secret/",
	"ldap":     "ldap/",
	"database": "database/",
}

func main() {
	// CLI flags
	duration := flag.Duration("duration", 60*time.Second, "Duration to run the simulation (e.g., 30s, 5m, 1h)")
	numRequests := flag.Int("num-requests", 100, "Number of requests to make during the duration")
	minInterval := flag.Duration("min-interval", 100*time.Millisecond, "Minimum interval between requests (e.g., 100ms, 1s)")
	maxInterval := flag.Duration("max-interval", 3*time.Second, "Maximum interval between requests (e.g., 1s, 10s)")
	requestTypes := flag.String("request-types", "kv", "Comma-delimited list of request types to simulate (accepted: kv, ldap, database)")

	flag.Parse()

	// Parse and validate request types
	types := parseRequestTypes(*requestTypes)
	if err := validateRequestTypes(types); err != nil {
		log.Fatal(err)
	}

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

	// Fail fast: check that required secret engines are mounted/enabled in Vault
	if err := checkMounts(client, types); err != nil {
		log.Fatalf("Mount check failed: %v", err)
	}

	// Fail fast: check for unimplemented request types
	checkUnimplemented(types)

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

	log.Printf("Starting vault-simulate: duration=%s, num-requests=%d, request-types=%s, vault-addr=%s (from VAULT_ADDR)",
		*duration, *numRequests, strings.Join(types, ","), config.Address)

	// Run the simulation
	if err := simulate(ctx, client, *duration, *numRequests, *minInterval, *maxInterval, types); err != nil {
		log.Fatalf("Simulation failed: %v", err)
	}

	log.Println("Simulation completed successfully")
}

// simulate runs the traffic simulation against Vault
func simulate(ctx context.Context, client *api.Client, duration time.Duration, numRequests int, minInterval, maxInterval time.Duration, requestTypes []string) error {
	intervals := generateRandomIntervals(duration, numRequests, minInterval, maxInterval)

	requestCount := 0
	successCount := 0
	errorCount := 0
	var createdKVKeys []string

	stopped := false
loop:
	for i, interval := range intervals {
		select {
		case <-ctx.Done():
			stopped = true
			break loop
		case <-time.After(interval):
			requestCount++
			key, err := makeVaultRequest(client, i)
			if err != nil {
				errorCount++
				log.Printf("Request %d failed: %v", requestCount, err)
			} else {
				successCount++
				createdKVKeys = append(createdKVKeys, key)
				log.Printf("Request %d completed successfully", requestCount)
			}
		}
	}

	if stopped {
		log.Printf("Simulation stopped: completed %d/%d requests (success: %d, errors: %d)",
			requestCount, numRequests, successCount, errorCount)
	} else {
		log.Printf("Simulation finished: completed %d/%d requests (success: %d, errors: %d)",
			requestCount, numRequests, successCount, errorCount)
	}

	// Clean up created KV keys
	if len(createdKVKeys) > 0 {
		cleanupKVKeys(client, createdKVKeys)
	}

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
func makeVaultRequest(client *api.Client, requestNum int) (string, error) {
	// Generate a unique key for this request
	key := fmt.Sprintf("simulate/key-%d-%d", requestNum, time.Now().UnixNano())
	secretData := map[string]interface{}{
		"value":     fmt.Sprintf("simulated-value-%d", requestNum),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Write a secret to KV v2
	_, err := client.KVv2("secret").Put(context.Background(), key, secretData)
	if err != nil {
		return "", fmt.Errorf("failed to write secret: %w", err)
	}

	return key, nil
}

// parseRequestTypes splits a comma-delimited string into a slice of trimmed, lowercased request types.
func parseRequestTypes(input string) []string {
	parts := strings.Split(input, ",")
	types := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(strings.ToLower(p))
		if t != "" {
			types = append(types, t)
		}
	}
	return types
}

// validateRequestTypes checks that all specified request types are recognized.
func validateRequestTypes(types []string) error {
	if len(types) == 0 {
		return fmt.Errorf("at least one request type must be specified")
	}
	for _, t := range types {
		if !validRequestTypes[t] {
			return fmt.Errorf("invalid request type %q: accepted types are kv, ldap, database", t)
		}
	}
	return nil
}

// checkMounts verifies that the required secret engines are mounted/enabled in Vault.
func checkMounts(client *api.Client, types []string) error {
	mounts, err := client.Sys().ListMounts()
	if err != nil {
		return fmt.Errorf("failed to list mounts: %w", err)
	}

	for _, t := range types {
		expectedPath, ok := mountPaths[t]
		if !ok {
			continue
		}
		if _, mounted := mounts[expectedPath]; !mounted {
			return fmt.Errorf("secret engine %q is not mounted at %q - please enable it before running the simulation", t, expectedPath)
		}
	}
	return nil
}

// checkUnimplemented fails fast if any of the specified request types are not yet implemented.
func checkUnimplemented(types []string) {
	for _, t := range types {
		switch t {
		case "ldap":
			// TODO: Implement LDAP secret engine simulation
			log.Fatalf("request type %q is not yet implemented", t)
		case "database":
			// TODO: Implement database secret engine simulation
			log.Fatalf("request type %q is not yet implemented", t)
		}
	}
}

// cleanupKVKeys deletes all KV keys created during the simulation.
func cleanupKVKeys(client *api.Client, keys []string) {
	log.Printf("Cleaning up %d KV keys...", len(keys))
	cleanupErrors := 0
	for _, key := range keys {
		err := client.KVv2("secret").DeleteMetadata(context.Background(), key)
		if err != nil {
			cleanupErrors++
			log.Printf("Failed to delete key %q: %v", key, err)
		}
	}
	if cleanupErrors > 0 {
		log.Printf("Cleanup completed with %d errors out of %d keys", cleanupErrors, len(keys))
	} else {
		log.Printf("Successfully cleaned up all %d KV keys from the simulation", len(keys))
	}
}
