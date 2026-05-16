# TODO

## Later Improvements

- Optional cheaper invalidation strategy, while keeping full polling as the correctness baseline.
- Broader Traefik label coverage for advanced service/router options.

## Architectural Follow-Up

- Remove the direct `proxmox/inventory` dependency on `traefik/labels`; keep inventory as a Proxmox snapshot builder and move Traefik enablement decisions behind an option, callback, or orchestration layer.
- Avoid parsing labels multiple times for the same workload. Aim for one authoritative parsed label representation that can answer both "enabled?" and config-building questions.
- Define a normalization boundary for generated Traefik object names and default host rules. Sanitize or reject unsafe PVE names with diagnostics, and keep raw workload names separate from generated Traefik names.
- Make provider publishing cancellation-safe by passing context into configuration publishing and selecting between `cfgChan <- payload` and `ctx.Done()`.
- Revisit the custom label AST/parser as label coverage grows. Consider a declarative supported-label path schema with special handlers only for shorthand, headers, and indexed TLS domains.
- Resolve the README/parser mismatch around Traefik fences: docs say only the first Traefik fence is parsed, while the parser currently scans and parses later Traefik fences too.
- Improve diagnostics for operators by preserving useful source context such as note line numbers or fragments when logging metadata/config problems.

## Performance Improvements

1. [x] Resolve IP addresses only for Traefik-enabled workloads.
2. [x] Apply `pve.requiredTags` before per-guest config and interface calls.
3. [x] Add bounded parallel scanning for per-node/per-guest API calls.
4. [x] Skip the `/nodes` API call when `pve.nodes` is explicitly configured.
5. [x] Investigate whether cluster-wide PVE resource endpoints can replace per-node VM/CT listing. `/cluster/resources?type=vm` can list cluster-wide VM/LXC resources with owning node information; implementation should wait until tags and permission behavior are validated against real fixtures.
6. [x] Precompute normalized node and required-tag filters when creating the scanner.

## Candidate Follow-Up

- Consider replacing per-node VM/CT list calls with `/cluster/resources?type=vm` after validating returned `tags`, `type`, `node`, `vmid`, `name`, and `status` fields across supported PVE versions.
