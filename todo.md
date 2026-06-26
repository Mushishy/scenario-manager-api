# Artemis - Project TODO

## Proposed Ludus Server Setup

* Newly purchased server running Ludus version 1.11.6 with Artemis
* Development server (upgrade the current Ludus server at 10.2.60.2 to the newest 2.x version). This server does not need to support Windows deployments; Linux only is sufficient for development.

## Guiding Principle

> If something can be done directly through the Ludus API, do it there.  
> The Scenario Manager API handles orchestration logic. The frontend handles UI, not business logic.  
> Read the entire TODO, including linked resources, before deciding what to work on first.

---

## Ready to Work On

### Backend

- [ ] **Document Ludus API calls that broke after the Ludus 2.x upgrade**  
  Track which existing Scenario Manager API or frontend calls no longer work against Ludus 2.x and need updating.

- [ ] **Migrate authentication to Pocketbase**  
  Ludus 2.x replaced SQLite with Pocketbase for its user database. The middleware auth check needs to use `FindFirstRecordByData` from Pocketbase instead of the old SQLite approach. Copy the pattern from the Ludus source code. If deploying to the school server, make sure to harden the Pocketbase instance.

```go
func APIKeyAuthenticationMiddleware(e *core.RequestEvent) error {
    record, err := e.App.FindFirstRecordByData("users", "userID", userID)
    // See (core.App).FindFirstRecordByData on pkg.go.dev
```

- [ ] **Replace topology management with Blueprints**  
  Ludus 2.x introduced Blueprints. Scenario Manager endpoints that manage topology files on disk can be removed. Endpoints that patch a topology for a range should be rewritten to apply a Blueprint instead.

- [ ] **Range creation: support groups of users**  
  In Ludus 2.x, ranges, users, and groups are more decoupled from each other.
  - For individual ranges, nothing changes. Every user has their own default range with access to its config, which can be modified via a file or Blueprint.
  - For shared ranges, decide whether to create a range as a separate entity tied to specific user IDs (user IDs can be added directly to the range during pool creation):
    ```bash
    ludus range create -r IDOWO -n OWO -d OWOUWU --users JSLIZIK,BATCHb
    ```
    or to create user groups to which ranges will be assigned. Both approaches work and reduce the number of individual share requests needed.

- [ ] **Replace manual Proxmox stats parsing with Ludus diagnostics**  
  Stop manually scraping Proxmox. Call `ludus diagnostics` directly instead.

- [ ] **Better file management (use a metafile)**  
  Replace file listing in the Scenario Manager API with a metafile that tracks files on disk. A metafile is a single index file that records what is stored, avoiding the need to scan the filesystem on every request. This reduces syscalls and speeds up file operations.

- [ ] **Fix CTFd data fetch: do not overwrite existing flags**  
  Fetching CTFd data into a pool currently overwrites any existing flags. For example, after enabling testing mode, log entries containing flags get wiped. Fix the fetch to merge or skip existing flags instead of replacing them. Additionally, the CTFd role currently requires flags to exist before deployment. Remove this check so that CTFd can be deployed without flags already being present.

- [ ] **Simplify BATCH prefix logic**  
  The BATCH prefix handling is spread across the frontend and backend. Simplify the logic on both sides.

- [ ] **Multiple simultaneous deployments: stress test and fix sleep timing**  
  Proxmox can create duplicate VM and network IDs when deploying multiple ranges at once, causing failures. Test and tune the sleep intervals between deployments.

- [ ] **Type-assertion panics throughout handlers**  
  Many handlers do `input["topologyId"].(string)` without the comma-ok idiom. In Go, a type assertion without the second return value will panic if the value is the wrong type. A malformed but schema-passing payload would crash the server. Use the safe form instead:
  ```go
  topologyId, ok := input["topologyId"].(string)
  if !ok { c.JSON(http.StatusBadRequest, ...); return }
  ```  

- [ ] **HTTP client is recreated on every request**  
  `createHTTPClient` in `http_helpers.go` creates a new `http.Transport` on every call. `http.Transport` is designed to be long-lived and reused across requests. Create one package-level client at startup.

- [ ] **No tests**  
  Neither the Go backend nor the Svelte frontend has any automated tests. At minimum, add unit tests for the most critical functions, and the auth flow.

### Frontend

- [ ] **Fix pool machine status in Artemis visualization**  
  The status indicator currently always shows machines as online. Fix it to reflect the actual state.

- [ ] **Testing Mode: granular IP and domain control**  
  Add more granular control over testing mode. Allow and deny specific IP addresses or domain names individually.

- [ ] **Fix UI for patching users: main user assignment**  
  The UI element for specifying the main user when patching should work the same way as when creating a pool.

- [ ] **Add link to observer management from the patch users page**  
  Observers are a feature most users are unaware of. Add a visible link from the patch users popup to observer management.

- [ ] **Fix input format for assigning users to teams and main users**  
  The input window could be larger and more flexible.

- [ ] **VPN download: include pool ID in filename**  
  WireGuard config downloads should include the pool ID in the filename to avoid confusion when managing multiple pools. Ludus also supports RDP/SSH config downloads; consider adding those too.

- [ ] **Add a sharing visualization page**  
  Add a new page that shows all currently shared ranges and who they are shared with. See [Ludus sharing docs](https://docs.ludus.cloud/docs/using-ludus/sharing).

- [ ] **Pool sharing and destroying**  
  Add color states for "all successfully unshared" and "all successfully destroyed." The pool destruction process should be more user-friendly, with a progress bar and clearer alert visibility.

- [ ] **Review and fix all Artemis action alerts**  
  Go through every alert and warning in the frontend and ensure they are accurate, non-redundant, and user-friendly.

- [ ] **Fix session expiration handling**  
  The current authentication logic is more complex than necessary. Review it, add longer expiry and refresh behavior, and test the changes thoroughly.

- [ ] **CTFd pools page: better filtering**  
  Add better filtering across all tables. For example, add an option to show only CTFd-relevant pools.

- [ ] **Error handling in client functions swallows context**  
  Every client function has `catch (error) { console.error(...); throw error; }`. The re-thrown error is the raw Axios error, which the caller (`pool-handlers.ts` -> `showAlert`) formats as a generic message. Parse `error.response.data` at the client layer and throw a typed `ApiError` with a human-readable `message` field.

- [ ] **Remove hardcoded dummy values on error**  
  When a request fails, the app falls back to fake hardcoded data. Remove this fallback and show an error message or an empty state instead.

- [ ] **Remove full page reload on login**  
  A full page reload is currently used to trigger the auth guard after login. Use `goto('/')` or `invalidateAll()` from SvelteKit instead, so navigation stays within the SPA and does not reset all in-flight requests.

- [ ] **Fix TypeScript errors**  
  `any` and `$derived` are used extensively throughout the client code. At minimum, the Axios response types should be parameterized.

---

## Needs Research First

These are good ideas but require investigation before implementation.

| # | Topic | Notes |
|---|-------|-------|
| 1 | **Ansible post-deployment button** | After a pool deploys, add an option to Ansible post-provision for time-based events (e.g., simulating defensive responses in a Red vs Blue game, or introducing new vulnerabilities). |
| 2 | **Ludus MCP** | [docs](https://docs.ludus.cloud/docs/using-ludus/mcp/) - Ludus has MCP support for the Ludus CLI. It can be used to build, create, and debug ranges. This could be a whole thesis project by itself. |
| 3 | **Notifications (Slack/Teams)** | Ludus supports `shoutrrr` in topology config. Research extending this to pool-level "all ranges deployed" notifications. |
| 4 | **Direct VM console links** | Similar to Proxmox's built-in UI, embed console links in the range graph view so users do not need to open Proxmox separately. |
| 5 | **Nexus cache** | If you are hitting Windows package rate limits on repeated deployments, [Nexus cache](https://docs.ludus.cloud/docs/infrastructure-operations/nexus-cache/) can help. |
| 6 | **File share** | To avoid re-downloading artifacts on repeated deploys, look into [Ludus file share](https://docs.ludus.cloud/docs/using-ludus/file-share). |
| 7 | **Unprivileged user access / CTFd plugin** | Allow students to revert their own range from CTFd by creating a custom CTFd plugin. This requires CTFd to have access to the Ludus API, which means adjusting the topology router iptables rules. |
| 8 | **Improved logging** | Log who did what and when in the Scenario Manager API. Currently `fmt.Println` and `log.Fatal` are used for all output. Add a structured logger (e.g., `slog` from the standard library, available since Go 1.21) with request IDs so log lines can be correlated to specific API calls. In the long term, Ludus will improve its own logging in future updates. What still needs research is in-range activity logging for individual users. |
| 9 | **Harden Ludus images** | Default passwords, SSH keys, and router credentials need to change. Blocker: Ansible cannot access machines after a credential change unless handled carefully. See [docs](https://docs.ludus.cloud/docs/using-ludus/passwords). |
| 10 | **Environment guides** | Test existing [Ludus environment guides](https://docs.ludus.cloud/docs/category/environment-guides) and new Ludus-specific roles for teaching value. |
| 11 | **Cluster support** | Investigate [Ludus cluster docs](https://docs.ludus.cloud/docs/infrastructure-operations/cluster) for scaling. |
| 12 | **CTFd Docker plugin** | Some exercises (binary exploitation, simple web challenges) do not need a full range. CTFd's Docker plugin, where each user gets their own containers, could serve them entirely. |
| 13 | **Snapshot management** | Expose Proxmox multi-snapshot management through the API and UI. See [Ludus snapshot docs](https://docs.ludus.cloud/docs/using-ludus/snapshots/). |
| 14 | **Autoshutdown support**  | When creating a range, provide an option to inject the autoshutdown config into the topology via the API. See [docs](https://docs.ludus.cloud/docs/enterprise/auto-shutdown). |
| 15 | **Multiple Routers** | Ludus supports only one router, and every VM has only one network interface. Additional interfaces can be added, and custom VMs with router-like capabilities can be provisioned to work around these limitations. Custom router images such as Security Onion could further improve this. |

---

## References

- [Ludus Docs](https://docs.ludus.cloud)
- [Ludus GitHub](https://github.com/orgs/badsectorlabs/repositories)
- [Ludus GitLab](https://gitlab.com/badsectorlabs)
- [Artemis Knowledge Base](https://gitlab.kypo.fiit.stuba.sk/stu-fiit-ludus/knowledge-base)
- [Ludus Enterprise WebUI](https://docs.ludus.cloud/docs/enterprise/webui): draw UI inspiration from here
