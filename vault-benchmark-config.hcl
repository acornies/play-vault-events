# Vault connection settings
vault_addr = "http://localhost:8200"
vault_token = "root"

# Benchmark duration (how long to generate traffic)
duration = "60s"

# Cleanup resources after benchmark completes
cleanup = true

# Random mount names to avoid conflicts
random_mounts = true

# KV v2 write test - generates continuous write traffic
test "kvv2_write" "kvv2_write_test" {
  weight = 100
  config {
    # Number of different keys to write to
    numkvs = 100
    
    # Size of each key/value in bytes
    kvsize = 100
  }
}
