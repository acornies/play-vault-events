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

## Generating Traffic with vault-simulate

The `vault-simulate` tool is a Go CLI application included in this repository that simulates organic traffic to a HashiCorp Vault server. It uses the official Vault Go SDK to make requests at random intervals, which is useful for generating events to visualize in real-time using event subscriptions or the Godot client.

### Building vault-simulate

Navigate to the `vault-simulate` directory and build the binary:

```bash
cd vault-simulate
go build -o vault-simulate .
```

### Usage

Run the simulator with the following options:

```bash
./vault-simulate -duration=60s -num-requests=100
```

#### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-duration` | Duration to run the simulation (e.g., `30s`, `5m`, `1h`) | `60s` |
| `-num-requests` | Number of requests to make during the duration | `100` |
| `-min-interval` | Minimum interval between requests (e.g., `100ms`, `1s`) | `100ms` |
| `-max-interval` | Maximum interval between requests (e.g., `1s`, `10s`) | `3s` |
| `-request-types` | Comma-delimited list of request types to simulate (accepted: `kv`, `ldap`, `database`) | `kv` |

#### Environment Variables

The Vault SDK automatically reads standard Vault environment variables:

| Variable | Description |
|----------|-------------|
| `VAULT_ADDR` | Vault server address (e.g., `http://localhost:8200`) |
| `VAULT_TOKEN` | Vault authentication token |
| `VAULT_CACERT` | Path to a CA certificate file (optional) |
| `VAULT_CLIENT_CERT` | Path to a client certificate file (optional) |
| `VAULT_CLIENT_KEY` | Path to a client key file (optional) |

See the [Vault SDK documentation](https://pkg.go.dev/github.com/hashicorp/vault/api#DefaultConfig) for a full list of supported environment variables.

#### Example Output

```
2024/02/04 12:00:00 Starting vault-simulate: duration=1m0s, num-requests=100, vault-addr=http://localhost:8200
2024/02/04 12:00:01 Request 1 completed successfully
2024/02/04 12:00:03 Request 2 completed successfully
...
2024/02/04 12:01:00 Simulation finished: completed 100/100 requests (success: 100, errors: 0)
2024/02/04 12:01:00 Simulation completed successfully
```

### How It Works

The simulator writes secrets to Vault's KV v2 secrets engine at the `secret/simulate/` path. Each request creates a unique key with a timestamp value. Requests are made at random intervals between `-min-interval` and `-max-interval` (defaulting to 100 milliseconds and 3 seconds) to simulate organic traffic patterns.

Use the `-request-types` flag to specify which secret engine types to simulate (e.g., `-request-types=kv,ldap,database`). The program will verify that the specified secret engines are mounted in Vault before starting, failing fast if any are missing. Currently, only the `kv` type is implemented; `ldap` and `database` are planned for future releases.

At the end of the simulation, all KV keys created during the run are automatically cleaned up.

This generates `kv-v2/data-write` events that you can observe through event subscriptions.

### Tips for Best Results

- Start event monitoring **before** running vault-simulate to catch all events
- Use the Godot client for a visual representation of the event stream
- Adjust `-duration`, `-num-requests`, `-min-interval`, and `-max-interval` based on how long you want to observe events and at what rate
- The tool can be stopped early with Ctrl+C for graceful shutdown

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