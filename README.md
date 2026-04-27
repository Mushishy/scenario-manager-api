# Scenario Manager API

> **A Go/Gin REST API backend for managing Ludus cybersecurity training range pools, topologies, and CTFd scenarios**

## Documentation

- **[Codebase Structure & Guide](./structure.md)** - Comprehensive guide to the repository structure, file organization, and core directories
- **[OpenAPI Specification](./openapi.yaml)** - Full API reference for all endpoints

## Overview

The Scenario Manager API acts as the central orchestration layer between the Artemis frontend, the Ludus range management system, and Proxmox. It handles pool lifecycle management, topology configuration, CTFd scenario deployment, and user operations — all authenticated via `X-API-Key` headers backed by bcrypt-hashed keys stored in the Ludus SQLite database.

## Quick Start

### Development

Generate a self-signed TLS certificate for local Proxmox communication:

```bash
cd server
mkdir certs
openssl req -x509 -newkey rsa:4096 -sha256 -nodes -days 3650 \
    -keyout ./certs/pve-ssl.key \
    -out ./certs/pve-ssl.pem \
    -subj "/C=SK/ST=Slovakia/L=Bratislava/O=STU/OU=ARTEMIS/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,DNS:*.localhost,IP:127.0.0.1,IP:::1"
```

Copy and edit the environment file:

```bash
cp .env.example .env
```

```
MAX_CONCURRENT_REQUESTS=4
DATA_LOCATION="./data"
DATABASE_LOCATION="/opt/ludus/ludus.db"
LUDUS_ADMIN_URL=https://localhost:8081
LUDUS_URL=https://localhost:8080
PROXMOX_URL=https://localhost:8006
PROXMOX_CERT_PATH="./certs"
PROXMOX_NODE_NAME=raven
DEPLOY_SLEEP_DURATION_SECONDS=3
```

Install dependencies and run:

```bash
go mod download
go mod tidy
go run .
```

When developing, forward Ludus ports locally via SSH and increase the file descriptor limit:

```
Host ludus
    HostName <hostname>
    User <username>
    IdentityFile ~/.ssh/<path to private key>
    LocalForward 8080 localhost:8080
    LocalForward 8081 localhost:8081
    LocalForward 8006 localhost:8006
```

```bash
ulimit -n 10000
```

To skip authentication during development, change the listen address to `127.0.0.1` in `main.go` and comment out `validateApiKey` in `routes.go` (this way you don't have to have the database locally).