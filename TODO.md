# TODO

## Current Status

We have the foundation in place:

- Traefik plugin catalog metadata and root plugin entrypoints.
- Proxmox API client for nodes, VMs, containers, config notes, VM agent interfaces, and LXC interfaces.
- Inventory scanner that builds typed VM/container snapshots.
- Fenced ` ```traefik ` metadata parsing by default, with `loose` and `auto` modes.
- Prefixless labels inside ` ```traefik ` fences, normalized to internal `traefik.*` labels.
- VM/container IP discovery.
- Node and required-tag scan filters for per-node or segmented Traefik instances.
- Offline Proxmox nodes are skipped during scans.
- Publish-on-change behavior using exact JSON payload comparison.
- Repeated scan/config diagnostics are de-duplicated across poll cycles to avoid log spam.
- Config-build diagnostics for unsupported labels, invalid bool/int values, and object name collisions.
- HTTP routers/services generated from `traefik.http.*` labels.
- Default HTTP router/service generation for enabled workloads.
- Routers prefer a same-named service by default, which supports multiple ports/services on one VM/container without explicit router service labels.
- Collision-safe default router/service naming when multiple workloads share a Proxmox name.
- Explicit duplicate HTTP services can merge compatible backend servers for clustered workloads.
- Backend URLs generated from explicit URL, explicit IP, discovered IPs, or hostname fallback.
- Multiple discovered IPs added as multiple load-balancer servers.
- Discovered IPs are sorted for stable output across polls.
- HTTP router options: rule, service binding, entryPoints, middlewares, priority, TLS, TLS cert resolver, TLS options, TLS domains.
- HTTP service options: port, scheme, URL, IP, passHostHeader, health checks, health-check headers, sticky sessions, response forwarding, serversTransport.
- Provider-managed HTTP servers transports for insecure skip verify, root CAs, max idle conns, HTTP/2 disabling, peer cert URI, and forwarding timeouts.
- Shorthand HTTP labels for the common path: `name`, `port`, `scheme`, `serverstransport`, `middlewares`, and `entrypoints` inside `traefik` fences.
- Optional `defaultDomain` support for generated HTTP host rules.
- TCP routers/services generated from `traefik.tcp.*` labels.
- UDP routers/services generated from `traefik.udp.*` labels.
- TCP backend addresses generated from explicit address, explicit IP, discovered IPs, or hostname fallback.
- UDP backend addresses generated from explicit address, explicit IP, discovered IPs, or hostname fallback.

## Parity Gap

The implementation is now beyond the original plugin for core HTTP behavior, diagnostics, TCP/UDP support, and note syntax. Remaining gaps are mostly around broader advanced Traefik option coverage and more real-world fixture coverage.

## Later Improvements

- Optional cheaper invalidation strategy, while keeping full polling as the correctness baseline.
- Broader Traefik label coverage for advanced service/router options.
