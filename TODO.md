# TODO

## Later Improvements

- Optional cheaper invalidation strategy, while keeping full polling as the correctness baseline.
- Broader Traefik label coverage for advanced service/router options.

## Performance Improvements

1. [x] Resolve IP addresses only for Traefik-enabled workloads.
2. [x] Apply `pve.requiredTags` before per-guest config and interface calls.
3. [ ] Add bounded parallel scanning for per-node/per-guest API calls.
4. [x] Skip the `/nodes` API call when `pve.nodes` is explicitly configured.
5. [ ] Investigate whether cluster-wide PVE resource endpoints can replace per-node VM/CT listing.
6. [x] Precompute normalized node and required-tag filters when creating the scanner.
