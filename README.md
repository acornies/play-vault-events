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

### 1. Subscribe to events using curl (HTTP streaming):

From a terminal, use curl with --no-buffer for streaming:

```bash
# Subscribe to KV v2 data write events
curl --no-buffer \
  --header "X-Vault-Token: root" \
  --request GET \
  http://localhost:8200/v1/sys/events/subscribe/kv-v2/data-write?json=true
```

This will open a long-lived HTTP connection that streams events as they occur.

### 2. Alternative: Subscribe using WebSocket:

You can also subscribe using the WebSocket protocol:

```bash
# Using websocat (install with: cargo install websocat or brew install websocat)
websocat "ws://localhost:8200/v1/sys/events/subscribe/kv-v2/data-write?json=true" -H="X-Vault-Token: root"
```

### 3. Generate events to observe:

In another terminal, perform some Vault operations to generate events:

```bash
# Create a secret
vault kv put secret/hello foo=world

# Read a secret
vault kv get secret/hello

# Delete a secret
vault kv delete secret/hello

# Create a policy
vault policy write my-policy - <<EOF
path "secret/data/*" {
  capabilities = ["create", "update"]
}
EOF

# List policies
vault policy list
```

### Event Types

Vault event types are organized by namespace. Common event types include:
- `kv-v2/data-write` - KV v2 secret write operations
- `kv-v2/data-read` - KV v2 secret read operations
- `kv-v2/data-delete` - KV v2 secret delete operations

For a complete list of event types, refer to the [Vault Events Documentation](https://developer.hashicorp.com/vault/docs/concepts/events).

### Advanced Event Subscription

Subscribe to multiple event types or use wildcards:

```bash
# Subscribe to all KV v2 events
curl --no-buffer \
  --header "X-Vault-Token: root" \
  --request GET \
  "http://localhost:8200/v1/sys/events/subscribe/kv-v2/*?json=true"
```

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