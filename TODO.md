# TODO

## Later Improvements

- Optional cheaper invalidation strategy, while keeping full polling as the correctness baseline.
- Broader Traefik label coverage for advanced service/router options.

## Performance Improvements

1. [x] Resolve IP addresses only for Traefik-enabled workloads.
2. [x] Apply `pve.requiredTags` before per-guest config and interface calls.
3. [x] Add bounded parallel scanning for per-node/per-guest API calls.
4. [x] Skip the `/nodes` API call when `pve.nodes` is explicitly configured.
5. [x] Investigate whether cluster-wide PVE resource endpoints can replace per-node VM/CT listing. `/cluster/resources?type=vm` can list cluster-wide VM/LXC resources with owning node information; implementation should wait until tags and permission behavior are validated against real fixtures.
6. [x] Precompute normalized node and required-tag filters when creating the scanner.

## Candidate Follow-Up

- Consider replacing per-node VM/CT list calls with `/cluster/resources?type=vm` after validating returned `tags`, `type`, `node`, `vmid`, `name`, and `status` fields across supported PVE versions.
