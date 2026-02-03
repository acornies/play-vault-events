# play-vault-events

This repository is for testing HashiCorp Vault Enterprise event notifications using Docker.

## Prerequisites

- Docker and Docker Compose installed on your system
- Basic knowledge of HashiCorp Vault

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

Vault provides event notifications that allow you to subscribe to various system events. Here's how to work with them:

### 1. Enable event streaming (if not already enabled in dev mode):

```bash
vault write sys/events/subscribe/audit events="*"
```

### 2. Subscribe to events using WebSocket:

You can subscribe to events using the WebSocket endpoint. From a separate terminal:

```bash
# Using websocat (install with: cargo install websocat or brew install websocat)
websocat "ws://localhost:8200/v1/sys/events/subscribe/audit?json=true" -H="X-Vault-Token: root"
```

Or using curl with --no-buffer for streaming:

```bash
curl --no-buffer \
  --header "X-Vault-Token: root" \
  --request GET \
  http://localhost:8200/v1/sys/events/subscribe/audit?json=true
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

Common Vault events you can subscribe to include:
- Authentication events
- Secret access events  
- Policy changes
- Token lifecycle events
- Audit log events

### Advanced Event Subscription

Subscribe to specific event types:

```bash
# Subscribe to only authentication events
curl --no-buffer \
  --header "X-Vault-Token: root" \
  --request GET \
  "http://localhost:8200/v1/sys/events/subscribe/auth?json=true"
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