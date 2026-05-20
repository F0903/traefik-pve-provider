# TODO

## Later Improvements

- Optional cheaper invalidation strategy, while keeping full polling as the correctness baseline.
- Broader Traefik label coverage for advanced service/router options.

## Interface-Scoped IP Resolution

Add a `pve.interfaces` metadata label that lets a workload opt into resolving
IPs only from explicitly selected guest interfaces. This is intended for cases
where otherwise-routable but undesired interfaces, such as `wg0` or
application-specific bridge interfaces like `pterodactyl0`, should not become
Traefik backend servers.

Proposed behavior:

- Label name: `pve.interfaces`.
- Accepted value: comma-separated interface patterns, for example
  `pve.interfaces=eth0`, `pve.interfaces=eth0,ens18`, or
  `pve.interfaces=eth*`.
- Matching: support shell-style globbing with `*` and `?`; exact names should
  continue to work as the simplest case.
- Scope: apply after Proxmox returns VM guest-agent or LXC interface data, and
  before IP mode filtering/deduplication.
- Default behavior: if `pve.interfaces` is absent, keep the current automatic
  behavior and built-in ignored-interface list.
- Override behavior: if `pve.interfaces` is present, only matching interfaces
  should be considered. The built-in ignored-interface list should still block
  obviously local/container bridge interfaces unless we intentionally add a
  second escape hatch later.
- Compatibility: this is provider metadata, not a Traefik dynamic-config label,
  so it should be consumed during `Prepare`/scan planning and not emitted into
  router or service configuration.

Implementation notes:

- Extend the label schema/parser to recognize `pve.interfaces` as a root
  provider option, likely stored on prepared workload label state rather than
  the Traefik resource set.
- Pass the selected interface patterns into `ResolveIPs` without making the
  scanner depend on Traefik label internals.
- Prefer `path.Match` or a small local glob matcher; verify with the Yaegi test
  before relying on it in runtime plugin code.
- Add focused tests for exact matches, comma-separated patterns, glob matches,
  no matches, default behavior, and interaction with `ipMode`.
- Extend the Yaegi regression test enough to cover parsing `pve.interfaces`.

