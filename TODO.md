# TODO

## Later Improvements

- Optional cheaper invalidation strategy, while keeping full polling as the correctness baseline.
- Broader Traefik label coverage for advanced service/router options.

## Architectural Follow-Up

- Remove the direct `proxmox/inventory` dependency on `traefik/labels`; keep inventory as a Proxmox snapshot builder and move Traefik enablement decisions behind an option, callback, or orchestration layer.
- Avoid parsing labels multiple times for the same workload. Aim for one authoritative parsed label representation that can answer both "enabled?" and config-building questions.
- Define a normalization boundary for generated Traefik object names and default host rules. Sanitize or reject unsafe PVE names with diagnostics, and keep raw workload names separate from generated Traefik names.
- Make provider publishing cancellation-safe by passing context into configuration publishing and selecting between `cfgChan <- payload` and `ctx.Done()`.
- Resolve the README/extractor mismatch around Traefik fences: docs say only the first Traefik fence is parsed, while the extractor currently scans and parses later Traefik fences too.
- Improve diagnostics for operators by preserving useful source context such as note line numbers or fragments when logging label extraction/config problems.

## Candidate Follow-Up

- Consider replacing per-node VM/CT list calls with `/cluster/resources?type=vm` after validating returned `tags`, `type`, `node`, `vmid`, `name`, and `status` fields across supported PVE versions.
