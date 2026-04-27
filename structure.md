## Project Structure

**Tech Stack:**
- **Framework:** Gin (Go)
- **Language:** Go 1.24
- **Database:** Ludus SQLite (via `modernc.org/sqlite`)
- **Authentication:** X-API-Key header with bcrypt-hashed keys
- **Config:** `.env` file loaded via `godotenv`
- **JSON Validation:** `gojsonschema`
- **Module name:** `dulus/server`

---

```
scenario-manager-api/
├── server/                                 # Go application source
│   ├── main.go                             # Entry point: DB init, SSL init, server start
│   ├── routes.go                           # Route registration & API key auth middleware
│   ├── go.mod                              # Go module definition and dependencies
│   │
│   ├── config/
│   │   └── config.go                       # Env-var loading (godotenv), all runtime config vars
│   │
│   ├── data/                               # Runtime data files (file-system persistence)
│   │   ├── ctfd_topology.yml               # Template CTFd topology (production)
│   │   └── topologies/
│   │       └── ctfdev/
│   │           └── ctfd_dev_topology.yml   # Template CTFd topology (dev)
│   │
│   ├── handlers/                           # Gin HTTP handler functions (one file per domain)
│   │   ├── ctfd_data_handler.go            # GET/PUT /ctfd/data, GET /ctfd/data/logins
│   │   ├── ctfd_scenario_handler.go        # GET/PUT/DELETE /ctfd/scenario
│   │   ├── ludus_range_config_handler.go   # POST/GET /range/config
│   │   ├── ludus_range_deploy_handler.go   # POST /range/deploy|redeploy|abort|remove, GET /range/status
│   │   ├── ludus_range_share_handler.go    # GET/POST /range/access|share|unshare|shared
│   │   ├── ludus_range_testing_handler.go  # PUT /range/testing/start|stop, GET /range/testing/status
│   │   ├── ludus_user_handler.go           # POST /users/import|delete, GET /users/check|main
│   │   ├── pool_handler.go                 # POST/GET/DELETE/PATCH /pool and /pool/dev
│   │   ├── proxmox_handler.go              # GET /stats/proxmox
│   │   └── topology_handler.go             # GET/PUT/DELETE /topology, POST /topology/ctfd
│   │
│   ├── schemas/                            # JSON Schema files for request body validation
│   │   ├── check_userids_schema.json
│   │   ├── ctfd_data_schema.json
│   │   ├── ctfd_topology_schema.json
│   │   ├── pool_note_schema.json
│   │   ├── pool_schema.json
│   │   ├── pool_topology_schema.json
│   │   └── pool_users_schema.json
│   │
│   └── utils/                              # Shared utility packages
│       ├── ctfd_operations.go              # CTFd topology generation, zip validation, data parsing
│       ├── deploy_state_manager.go         # In-memory deploying-pool state (mutex-guarded map)
│       ├── file_operations.go              # File read/write helpers, ID generation, dir utilities
│       ├── function_helpers.go             # bcrypt hashing, random strings, JSON schema validation
│       ├── http_helpers.go                 # Query param helpers, HTTP client factory, response converters
│       ├── ludus_client.go                 # Ludus API HTTP client, concurrent request dispatcher, Pool/RangeStatus types
│       ├── pool_operations.go              # Pool JSON read/write, user ID extraction from pool
│       ├── proxmox_operations.go           # Proxmox API client, statistics aggregation
│       └── users_operations.go             # User/team validation, special-char normalization, Ludus user ops
│
├── build.sh                                # Build script
├── openapi.yaml                            # OpenAPI API specification
├── scenario-manager-api.service            # Systemd service unit for production deployment
├── README.md                               # Project overview & setup
├── structure.md                            # This file - codebase structure
└── todo.md                                 # Development task list
```

---

## Core Directories

### `server/config`
**Purpose:** Centralised runtime configuration

- Loads all environment variables from `.env` at startup via `godotenv`
- Exports typed Go vars consumed across the whole codebase:
  - `LudusAdminUrl`, `LudusUrl` — Ludus API base URLs
  - `ProxmoxURL`, `ProxmoxCertPath`, `ProxmoxNodeName` — Proxmox connection
  - `DatabaseLocation` — SQLite file path
  - `CtfdScenarioFolder`, `TopologyConfigFolder`, `PoolFolder` — file-system data paths
  - `MaxConcurrentRequests`, `DeploySleepDuration` — concurrency tuning

### `server/handlers`
**Purpose:** Thin Gin handler layer — validates input, delegates to utils, returns JSON

| File | Routes covered |
|------|---------------|
| `ctfd_scenario_handler.go` | `GET/PUT/DELETE /ctfd/scenario` |
| `ctfd_data_handler.go` | `GET/PUT /ctfd/data`, `GET /ctfd/data/logins` |
| `topology_handler.go` | `GET/PUT/DELETE /topology`, `POST /topology/ctfd` |
| `pool_handler.go` | `POST/GET/DELETE /pool`, `POST /pool/dev`, `PATCH /pool/topology|note|users`, `POST /pool/users` |
| `ludus_user_handler.go` | `POST /users/import|delete`, `GET /users/check|main` |
| `ludus_range_config_handler.go` | `POST/GET /range/config` |
| `ludus_range_deploy_handler.go` | `POST /range/deploy|redeploy|abort|remove`, `GET /range/status` |
| `ludus_range_share_handler.go` | `GET /range/access|shared|shared/user`, `POST /range/share|unshare|share/user|unshare/user` |
| `ludus_range_testing_handler.go` | `PUT /range/testing/start|stop`, `GET /range/testing/status` |
| `proxmox_handler.go` | `GET /stats/proxmox` |

### `server/utils`
**Purpose:** Shared business logic and infrastructure helpers

- **`ludus_client.go`** — HTTP client for the Ludus API; concurrent fan-out dispatcher (`MakeConcurrentLudusRequests`); defines `Pool`, `RangeStatus`, `UserTeam` types
- **`pool_operations.go`** — Read/write `pool.json` files; extract user IDs from a pool by retrieval mode (`SharedMainUserOnly`, `SharedUsersAndTeamsOnly`, `SharedAllUsers`)
- **`deploy_state_manager.go`** — Thread-safe in-memory set that tracks which pools are currently deploying; prevents duplicate deployments
- **`ctfd_operations.go`** — Generates CTFd Ludus topology YAMLs from templates; validates and inspects CTFd scenario zip archives; parses CTFd login data
- **`file_operations.go`** — Directory/file helpers: read first file in dir, save uploaded files, `EnsureDirectoryExists`, `ValidateFolderId`
- **`function_helpers.go`** — `GenerateUniqueID`, random strings, bcrypt hash/verify, JSON schema validation via `gojsonschema`, `ExtractUserIDFromAPIKey`
- **`http_helpers.go`** — `GetRequiredQueryParam`, `GetOptionalQueryParam`, TLS-skip HTTP client, `ConvertResponsesToResults`
- **`proxmox_operations.go`** — Proxmox REST client; authenticates with ticket/CSRF; aggregates cluster resource statistics
- **`users_operations.go`** — Validates and processes `usersAndTeams` arrays; normalises special characters in usernames; maps Ludus user operations

### `server/schemas`
**Purpose:** JSON Schema files used by `ValidateJSONSchema` to validate request bodies before processing

| File | Used by |
|------|---------|
| `pool_schema.json` | `POST /pool` |
| `pool_topology_schema.json` | `PATCH /pool/topology` |
| `pool_note_schema.json` | `PATCH /pool/note` |
| `pool_users_schema.json` | `PATCH /pool/users` |
| `check_userids_schema.json` | `POST /pool/users` (check) |
| `ctfd_data_schema.json` | `PUT /ctfd/data` |
| `ctfd_topology_schema.json` | `POST /topology/ctfd` |

### `server/data`
**Purpose:** File-system data store for persistent objects

- `ctfd_topology.yml` — Master Ludus topology template for CTFd production deployments
- `topologies/` — User-uploaded topology YAML files (each in its own ID-named subdirectory)
- `ctfd_scenarios/` *(runtime)* — Uploaded CTFd scenario zip files
- `pools/` *(runtime)* — Pool JSON files (`pool.json`) and associated CTFd data

---

## Authentication

All routes (except `GET /`) require an `X-API-Key` header.

The `validateAPIKey` middleware in `routes.go`:
1. Extracts the user ID embedded in the API key
2. Looks up the bcrypt-hashed key from SQLite (`user_objects` table)
3. Verifies the key and sets `userID` and `isAdmin` in the Gin context
4. Implements exponential-backoff retry for SQLite busy errors

---

## API Route Summary

| Group | Endpoints |
|-------|-----------|
| **CTFd Scenario** | `GET/PUT/DELETE /ctfd/scenario` |
| **CTFd Data** | `GET/PUT /ctfd/data`, `GET /ctfd/data/logins` |
| **Topology** | `GET/PUT/DELETE /topology`, `POST /topology/ctfd` |
| **Pool** | `POST/GET/DELETE /pool`, `POST /pool/dev`, `PATCH /pool/topology\|note\|users`, `POST /pool/users` |
| **Users** | `POST /users/import\|delete`, `GET /users/check\|main` |
| **Range Config** | `POST/GET /range/config` |
| **Range Deploy** | `POST /range/deploy\|redeploy\|abort\|remove`, `GET /range/status` |
| **Range Share** | `GET/POST /range/access\|share\|unshare\|shared\|shared/user\|share/user\|unshare/user` |
| **Range Testing** | `PUT /range/testing/start\|stop`, `GET /range/testing/status` |
| **Statistics** | `GET /stats/proxmox` |

---

## Configuration Files

| File | Purpose |
|------|---------|
| **scenario-manager-api.service** | Systemd service unit for production deployment |
| **build.sh** | Builds the Go binary |
| **openapi.yaml** | OpenAPI 3.x specification for all endpoints |
| **README.md** | Project overview & setup instructions |
| **server/go.mod** | Go module definition and dependency versions |

