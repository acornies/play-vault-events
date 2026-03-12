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
// The "kv" type is excluded because its mount path is configurable via the -kv-path flag.
var mountPaths = map[string]string{
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
	kvPath := flag.String("kv-path", "simulate-secret", "Vault KV v2 mount path used for the simulation")

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

	// Fail fast: check that required secret engines (other than kv) are mounted/enabled in Vault
	if err := checkMounts(client, types); err != nil {
		log.Fatalf("Mount check failed: %v", err)
	}

	// Fail fast: check for unimplemented request types
	checkUnimplemented(types)

	// Enable KV v2 mount at kv-path if kv type is requested
	if containsType(types, "kv") {
		if err := enableKVMount(client, *kvPath); err != nil {
			log.Fatalf("Failed to enable KV mount at %q: %v", *kvPath, err)
		}
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

	log.Printf("Starting vault-simulate: duration=%s, num-requests=%d, request-types=%s, kv-path=%s, vault-addr=%s (from VAULT_ADDR)",
		*duration, *numRequests, strings.Join(types, ","), *kvPath, config.Address)

	// Run the simulation
	if err := simulate(ctx, client, *duration, *numRequests, *minInterval, *maxInterval, types, *kvPath); err != nil {
		log.Fatalf("Simulation failed: %v", err)
	}

	log.Println("Simulation completed successfully")
}

// simulate runs the traffic simulation against Vault
func simulate(ctx context.Context, client *api.Client, duration time.Duration, numRequests int, minInterval, maxInterval time.Duration, requestTypes []string, kvPath string) error {
	intervals := generateRandomIntervals(duration, numRequests, minInterval, maxInterval)

	requestCount := 0
	successCount := 0
	errorCount := 0

	stoppedEarly := false
loop:
	for i, interval := range intervals {
		select {
		case <-ctx.Done():
			stoppedEarly = true
			break loop
		case <-time.After(interval):
			requestCount++
			if err := makeVaultRequest(client, i, kvPath); err != nil {
				errorCount++
				log.Printf("Request %d failed: %v", requestCount, err)
			} else {
				successCount++
				log.Printf("Request %d completed successfully", requestCount)
			}
		}
	}

	if stoppedEarly {
		log.Printf("Simulation stopped: completed %d/%d requests (success: %d, errors: %d)",
			requestCount, numRequests, successCount, errorCount)
	} else {
		log.Printf("Simulation finished: completed %d/%d requests (success: %d, errors: %d)",
			requestCount, numRequests, successCount, errorCount)
	}

	// Clean up by disabling and re-enabling the KV mount
	if containsType(requestTypes, "kv") {
		cleanupKVMount(client, kvPath)
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
func makeVaultRequest(client *api.Client, requestNum int, kvPath string) error {
	// Generate a unique key for this request
	key := fmt.Sprintf("simulate/key-%d-%d", requestNum, time.Now().UnixNano())
	secretData := map[string]interface{}{
		"value":     fmt.Sprintf("simulated-value-%d", requestNum),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Write a secret to KV v2
	_, err := client.KVv2(kvPath).Put(context.Background(), key, secretData)
	if err != nil {
		return fmt.Errorf("failed to write secret: %w", err)
	}

	return nil
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
// The "kv" type is excluded because its mount is managed by the -kv-path flag.
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

// containsType checks if a slice of request types contains a specific type.
func containsType(types []string, target string) bool {
	for _, t := range types {
		if t == target {
			return true
		}
	}
	return false
}

// enableKVMount enables a KV v2 secrets engine at the specified path.
// If the mount already exists, it is left as-is.
func enableKVMount(client *api.Client, kvPath string) error {
	mounts, err := client.Sys().ListMounts()
	if err != nil {
		return fmt.Errorf("failed to list mounts: %w", err)
	}

	mountKey := kvPath + "/"
	if _, exists := mounts[mountKey]; exists {
		log.Printf("KV v2 secrets engine already mounted at %q", kvPath)
		return nil
	}

	mountInput := &api.MountInput{
		Type:    "kv",
		Options: map[string]string{"version": "2"},
	}
	if err := client.Sys().Mount(kvPath, mountInput); err != nil {
		return fmt.Errorf("failed to enable KV v2 at %q: %w", kvPath, err)
	}
	log.Printf("Enabled KV v2 secrets engine at %q", kvPath)
	return nil
}

// cleanupKVMount cleans up the KV mount by disabling and re-enabling it.
// This effectively removes all secrets without deleting them individually.
func cleanupKVMount(client *api.Client, kvPath string) {
	log.Printf("Cleaning up KV mount at %q...", kvPath)

	// Use a separate timeout context for cleanup to avoid hanging indefinitely
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cleanupCancel()

	// Disable the mount (removes all data)
	if err := client.Sys().UnmountWithContext(cleanupCtx, kvPath); err != nil {
		log.Printf("Failed to disable KV mount at %q: %v", kvPath, err)
		return
	}

	// Re-enable the mount
	mountInput := &api.MountInput{
		Type:    "kv",
		Options: map[string]string{"version": "2"},
	}
	if err := client.Sys().MountWithContext(cleanupCtx, kvPath, mountInput); err != nil {
		log.Printf("Failed to re-enable KV mount at %q: %v", kvPath, err)
		return
	}

	log.Printf("Successfully cleaned up KV mount at %q", kvPath)
}
