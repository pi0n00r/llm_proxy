# Docker Setup for LLM Proxy

This guide explains how to run the LLM proxy server using Docker.

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 1.29+ (optional, for orchestration)

## Image Status

Versioned images for this fork are published at
`ghcr.io/pi0n00r/llm_proxy`. Prefer an immutable version tag such as
`v0.2.23` rather than `latest`.

The image runs as unprivileged UID/GID `10001:10001`. A Docker named volume is
the simplest persistent-storage option because Docker initializes it with the
image directory's ownership. For a bind mount, make the host data directory
writable by UID 10001 before starting the container.

## Run the Published Image

```bash
docker pull ghcr.io/pi0n00r/llm_proxy:v0.2.23
docker run -d \
  --name llm-proxy \
  -p 6666:6666 \
  -v "$(pwd)/config.toml:/app/config/config.toml:ro" \
  -v llm-proxy-data:/app/data \
  ghcr.io/pi0n00r/llm_proxy:v0.2.23
```

## Building from Source

If you prefer to build the Docker image from source (for development or customization):

### Quick Start

### 1. Prepare Configuration

First, create your `config.toml` from the example:

```bash
cp config.toml.example config.toml
```

The default `database.path = "./data/llm_proxy.db"` resolves to `/app/data/llm_proxy.db` because the image runs from `/app`. Edit `backend.endpoint` for wherever your backend is reachable; a backend running on the Docker host will generally need `host.docker.internal` instead of `localhost`.

### 2. Create Docker Compose Override

The base `docker-compose.yml` has ports and volumes commented out. Create an override file:

```bash
cp docker-compose.override.yml.example docker-compose.override.yml
```

This file enables:
- Port mapping on compatibility port `6666`
- Config file volume mount
- Database persistence directory

Edit `docker-compose.override.yml` if you need different port mappings or paths.

### 3. Create Data Directory

The database will be stored in the `data/` directory:

```bash
mkdir -p data
sudo chown 10001:10001 data
```

### 4. Build and Run

Using Docker Compose (recommended):

```bash
docker-compose up -d
```

Or manually with Docker:

```bash
# Build the image
docker build -t llm-proxy .

# Run the container
docker run -d \
  --name llm-proxy \
  -p 6666:6666 \
  -v "$(pwd)/config.toml:/app/config/config.toml:ro" \
  -v "$(pwd)/data:/app/data" \
  llm-proxy
```

**Note**: The compatibility fork uses reserved proxy port `6666`, leaving Ollama's default port untouched.

### 5. Verify It's Running

```bash
# Check container status
docker-compose ps

# Check logs
docker-compose logs -f

# Test the endpoint
curl http://localhost:6666/
# Should show the web UI HTML

# Health check
curl http://localhost:6666/health
# Should return: "OK"

# List models
curl http://localhost:6666/api/tags
```

## Configuration

### Docker Compose Structure

This project uses a layered Docker Compose approach:

1. **`docker-compose.yml`** (base configuration)
   - Defines the service build and health checks
   - Ports and volumes are commented out
   - Committed to version control

2. **`docker-compose.override.yml`** (your local configuration)
   - Extends the base configuration
   - Sets up ports and volume mounts
   - **Not committed** to version control (in `.gitignore`)
   - Create from `docker-compose.override.yml.example`

This approach allows each deployment to customize ports and paths without modifying tracked files.

### Volume Mounts

The `docker-compose.override.yml` sets up two volume mounts:

1. **Config file** (`./config.toml` → `/app/config/config.toml`)
   - Mounted as read-only (`:ro`)
   - Contains server and backend configuration
   - Edit this file to change proxy settings
   - Use `config.toml.example` as a template

2. **Database directory** (`./data` → `/app/data`)
   - Stores the SQLite database (`llm_proxy.db`)
   - Persists request/response logs across container restarts
   - The example's relative path, `"./data/llm_proxy.db"`, resolves to this mounted directory inside the container

### Environment Variables

You can customize the timezone or other settings in `docker-compose.yml`:

```yaml
environment:
  TZ: Europe/London  # Change to your timezone
```

## Managing the Container

### Start/Stop

```bash
# Start
docker-compose up -d

# Stop
docker-compose down

# Restart
docker-compose restart
```

### View Logs

```bash
# Follow logs
docker-compose logs -f

# View last 100 lines
docker-compose logs --tail=100
```

### Update Configuration

After editing `config.toml`:

```bash
docker-compose restart
```

### Rebuild After Code Changes

```bash
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

## Troubleshooting

### Cannot connect to backend

If the backend is running on your host machine:
- Use `host.docker.internal` instead of `localhost` in your `backend.endpoint`
- Example: `endpoint = "http://host.docker.internal:8008"`
- The example uses `localhost`, so change it when the backend runs on the Docker host

On Linux, you may need to add this to your `docker-compose.override.yml`:

```yaml
services:
  llm-proxy:
    extra_hosts:
      - "host.docker.internal:host-gateway"
```

### Cannot access the proxy

If you can't reach the proxy from your browser or Home Assistant:
- Check that `docker-compose.override.yml` exists and has the ports section
- Verify the port mapping: `docker-compose ps`
- Ensure the config has `host = "::"` (already set in `config.toml.example`)
- Check container logs: `docker-compose logs -f`

### Database permission errors

Ensure the `data/` directory is writable:

```bash
chmod 755 data/
```

### Port already in use

The default `docker-compose.override.yml.example` uses port `6666`.

If you need a different port, edit your `docker-compose.override.yml`:

```yaml
services:
  llm-proxy:
    ports:
      - "11436:6666"  # Use external port 11436
```

## Production Considerations

### Resource Limits

Add resource limits in docker-compose.yml:

```yaml
deploy:
  resources:
    limits:
      cpus: '2'
      memory: 1G
    reservations:
      cpus: '0.5'
      memory: 256M
```

### Logging

Configure log rotation in docker-compose.yml:

```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

### Backup Database

```bash
# Backup
docker-compose exec llm-proxy cp /app/data/llm_proxy.db /app/data/llm_proxy.db.backup

# Or from host
cp data/llm_proxy.db data/llm_proxy.db.backup
```

## Advanced Usage

### Multiple Configurations

You can maintain multiple configuration files for different backends:

```bash
# For llama.cpp backend
cp config.toml.example config_llama_cpp.toml

# For Ollama backend
cp config.toml.example config_ollama.toml
# Edit config_ollama.toml to set type = "ollama"
```

Then reference the desired config in your `docker-compose.override.yml`:

```yaml
services:
  llm-proxy:
    volumes:
      - ./config_llama_cpp.toml:/app/config/config.toml:ro
      - ./data:/app/data
```

### Running with custom network

If you have other services (like a local LLM backend) in Docker, add a network configuration to `docker-compose.override.yml`:

```yaml
services:
  llm-proxy:
    networks:
      - llm-network

networks:
  llm-network:
    external: true
    name: my-existing-network
```

### Accessing the Web UI

The proxy includes a built-in web interface:

- **Home page**: `http://localhost:6666/` - Shows configuration overview
- **Logs**: `http://localhost:6666/logs` - Browse all requests
- **Details**: `http://localhost:6666/logs/details?id=<id>` - View specific request details

Replace `6666` with your configured port.

### Multi-stage debugging

Build without cache and see all output:

```bash
docker build --no-cache --progress=plain -t llm-proxy .
```
