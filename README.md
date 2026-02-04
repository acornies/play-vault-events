# play-vault-events

This repository is for testing HashiCorp Vault Enterprise event notifications using Docker.

**Note**: Event notifications are an **Enterprise feature** available in Vault v1.21+. You will need a valid Vault Enterprise license to use this feature.

## Prerequisites

- Docker and Docker Compose installed on your system
- Basic knowledge of HashiCorp Vault
- A valid Vault Enterprise license

## Setup

Before running Vault, you need to set your Vault Enterprise license as an environment variable:

```bash
export VAULT_LICENSE="your-license-key-here"
```

Alternatively, create a `.env` file in the repository root with your license:

```
VAULT_LICENSE=your-license-key-here
```

Docker Compose will automatically load the `.env` file.

## Running Vault with Docker Compose

1. **Start the Vault container:**

   ```bash
   docker compose up -d
   ```

   Note: If using Docker Compose v1, use `docker-compose up -d` instead.

   This will start Vault in development mode on `http://localhost:8200` with the root token set to `root`.

2. **Verify Vault is running:**

   ```bash
   docker compose ps
   ```

3. **Check Vault logs:**

   ```bash
   docker compose logs -f vault
   ```

## Using Vault CLI

To interact with Vault, you can use the Vault CLI from within the container or install it locally.

### Using Vault CLI from the container:

```bash
docker exec -it vault sh
```

Once inside the container, set the Vault address and token:

```bash
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='root'
```

### Using Vault CLI locally:

If you have Vault CLI installed on your host machine:

```bash
export VAULT_ADDR='http://localhost:8200'
export VAULT_TOKEN='root'
```

## Subscribing to Vault Event Notifications

Vault provides event notifications that allow you to subscribe to various system events. Events are consumed via HTTP streaming or WebSocket connections.

### 1. Subscribe to events using Vault CLI:

The simplest way to subscribe to events is using the Vault CLI:

```bash
# Subscribe to all KV v2 data events
vault events subscribe kv-v2/data-*
```

### 2. Subscribe to events using curl (HTTP streaming):

From a terminal, use curl with --no-buffer for streaming:

```bash
# Subscribe to all KV v2 data events
curl --no-buffer \
  --header "X-Vault-Token: root" \
  --request GET \
  http://localhost:8200/v1/sys/events/subscribe/kv-v2/data-*?json=true
```

This will open a long-lived HTTP connection that streams events as they occur.

### 3. Alternative: Subscribe using WebSocket:

You can also subscribe using the WebSocket protocol:

```bash
# Using websocat (install with: cargo install websocat or brew install websocat)
websocat "ws://localhost:8200/v1/sys/events/subscribe/kv-v2/data-*?json=true" -H="X-Vault-Token: root"
```

### 4. Generate events to observe:

In another terminal, perform some Vault operations to generate events:

```bash
# Create a secret
vault kv put secret/hello foo=world

# Read a secret
vault kv get secret/hello

# Delete a secret
vault kv delete secret/hello
```

### Event Types

For a complete list of event types, refer to the [Vault Events Documentation](https://developer.hashicorp.com/vault/docs/concepts/events).

## Generating Traffic with vault-benchmark

[vault-benchmark](https://github.com/hashicorp/vault-benchmark) is a performance testing tool that can simulate realistic traffic to Vault. This is especially useful for generating a continuous stream of events to visualize in real-time using the event subscriptions or the Godot client.

### Installing vault-benchmark

#### Option 1: Download Release Binary

Download the latest release from [HashiCorp releases](https://releases.hashicorp.com/vault-benchmark):

```bash
# Example for Linux AMD64 (adjust for your platform)
wget https://releases.hashicorp.com/vault-benchmark/<VERSION>/vault-benchmark_<VERSION>_linux_amd64.zip
unzip vault-benchmark_<VERSION>_linux_amd64.zip
chmod +x vault-benchmark
sudo mv vault-benchmark /usr/local/bin/
```

#### Option 2: Build from Source

If you have Go installed:

```bash
git clone https://github.com/hashicorp/vault-benchmark.git
cd vault-benchmark
make bin
# Binary will be in dist/<OS>/<ARCH>/vault-benchmark
```

### Using kvv2_write_test Benchmark

The `kvv2_write_test` benchmark continuously writes to KV v2 secrets, which generates `kv-v2/data-write` events that you can observe through event subscriptions.

#### 1. Create a Benchmark Configuration File

A sample configuration file `vault-benchmark-config.hcl` is provided in this repository:

```hcl
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
```

#### 2. Run the Benchmark

```bash
vault-benchmark run -config=vault-benchmark-config.hcl
```

You should see output similar to:

```
2024-02-04T12:00:00.000-0000 [INFO]  vault-benchmark: setting up targets
2024-02-04T12:00:02.000-0000 [INFO]  vault-benchmark: starting benchmarks: duration=60s
2024-02-04T12:01:02.000-0000 [INFO]  vault-benchmark: cleaning up targets
2024-02-04T12:01:05.000-0000 [INFO]  vault-benchmark: benchmark complete

Target: http://localhost:8200
op                count   rate        throughput  mean      95th%     99th%     successRatio
kvv2_write_test   30234   503.906667  503.750000  1.98ms    3.12ms    4.56ms    100.00%
```

#### Configuration Options

You can customize the benchmark behavior:

- **duration**: How long to run the test (e.g., `"30s"`, `"5m"`, `"1h"`)
- **numkvs**: Number of unique keys to write (more keys = more varied paths)
- **kvsize**: Size of the data written to each secret (in bytes)
- **weight**: If running multiple test types, determines the percentage split (100 = 100%)

#### Advanced: Multiple Test Types

You can run multiple test types simultaneously:

```hcl
vault_addr = "http://localhost:8200"
vault_token = "root"
duration = "60s"
cleanup = true
random_mounts = true

# 70% of traffic will be writes
test "kvv2_write" "kvv2_write_test" {
  weight = 70
  config {
    numkvs = 100
    kvsize = 100
  }
}

# 30% of traffic will be reads
test "kvv2_read" "kvv2_read_test" {
  weight = 30
  config {
    numkvs = 100
  }
}
```

### Tips for Best Results

- Start event monitoring **before** running vault-benchmark to catch all events
- Use the Godot client for a visual representation of the event stream
- Adjust `duration` based on how long you want to observe events
- Higher `numkvs` values create more diverse event data
- The `cleanup = true` setting removes test data after completion

## Godot WebSocket Client

This repository includes a Godot Engine project (`godot/`) that demonstrates consuming Vault enterprise events through a WebSocket connection in real-time.

### What it does

The Godot project provides an interactive, visual way to monitor Vault events as they occur. The main script ([main.gd](godot/main.gd)) connects to Vault's WebSocket endpoint and displays incoming event notifications in the Godot console.

Key features:
- **Real-time event streaming**: Connects via WebSocket to `ws://localhost:8200` 
- **Authentication**: Uses Vault token authentication via `X-Vault-Token` header
- **Event display**: Prints all received Vault events to the Godot debug console
- **Connection management**: Handles WebSocket connection states (connecting, open, closing, closed)

### Running the Godot client

1. **Install Godot Engine**: Download from [godotengine.org](https://godotengine.org/)

2. **Set up Vault token**: In the Godot editor, select the main scene and set the `auth_token` export variable to your Vault token (default: `root`)

3. **Run the project**: Press F5 in Godot or click the play button

4. **Monitor events**: The Godot console will display incoming Vault events in real-time as you perform operations

This provides an alternative to command-line tools like curl or websocat, especially useful for:
- Visual debugging of event flows
- Interactive demonstrations of Vault events
- Game development scenarios where Vault events need to trigger in-game actions
- Learning and experimenting with Vault's event streaming capabilities

**Note**: The Godot client requires Vault to be running and accessible at `http://localhost:8200`.

## Stopping the Container

To stop and remove the Vault container:

```bash
docker compose down
```

To stop without removing (data will persist):

```bash
docker compose stop
```

## Notes

- This setup uses Vault in **development mode**, which is **NOT suitable for production**.
- In dev mode, data is stored in-memory and will be lost when the container is stopped.
- The root token is hardcoded as `root` for convenience in testing.
- For production deployments, refer to the official Vault documentation: https://developer.hashicorp.com/vault/docs

## Additional Resources

- [Vault Events Documentation](https://developer.hashicorp.com/vault/docs/concepts/events)
- [Vault Docker Hub](https://hub.docker.com/r/hashicorp/vault)
- [Vault Getting Started Guide](https://developer.hashicorp.com/vault/tutorials/getting-started)