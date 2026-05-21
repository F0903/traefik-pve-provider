# Traefik PVE Provider

Automatically create Traefik routers and services via your PVE container/VM notes.

Inspired by [traefik-proxmox-provider](https://github.com/NX211/traefik-proxmox-provider) but with simpler configuration and a larger feature-set.
> Note that this is not a fork, but a project built from scratch.

## Traefik Installation

To install the plugin, add the following two blocks to your Traefik configuration.

First, define the plugin with the name and version as follows:

> This **needs** to go in your static Traefik configuration. *(usually called `traefik.yml`)*

```yaml
experimental:
  plugins:
    traefik-pve-provider:
      moduleName: github.com/F0903/traefik-pve-provider
      version: v0.11.0
```

Secondly, define the plugin as a provider along with your preferred settings:

> This can either go in your static or dynamic Traefik configuration.

Also note that the following is an example, some of the settings are optional and can be omitted.

```yaml
providers:
  plugin:
    traefik-pve-provider:
      pollInterval: 60s
      metadataMode: fenced
      defaultDomain: example.com
      pve:
        endpoint: "https://pve.example.com"
        tokenID: "your-token-id"
        token: "your-token-secret"
        timeout: "5s"
        insecureSkipVerify: true
        skipStopped: true
        skipIPResolution: false
        ipMode: ipv4
        defaultInterfaces:
          - eth*
          - enp*
          - eno*
        maxConcurrency: 4
        nodes:
          - pve-1
        requiredTags:
          - traefik
```

## Usage

Depending on the plugin configuration, there are two primary ways to configure the services and routers:

### Traefik Fences

The default and preferred method is using "traefik fences". 
These are Markdown fenced codeblocks that contain "traefik" in the info string.

In these fences, you can omit the Traefik prefix of the configuration labels, and use simpler versions of some labels.

An additional benefit is that the service name is automatically derived from the VM/container name, further reducing repetition.
If this is not desired, you can manually overwrite it with `name=<name>`.

Only the first "traefik fence" inside a note is parsed, and the rest of the note is ignored.
Additionally, blank lines and lines starting with `#` are ignored.
Every other line must use `key=value`. Prefixless keys are normalized as
Traefik labels, while loose mode still requires explicit `traefik.*` labels.

Example inside VM/container notes:

````md
```traefik
enable=true
port=8080
middlewares=local-only@file,compress-all@file
```
````

Additionally, with `defaultDomain` configured, an enabled workload will be automatically assigned a router and service with a `Host` rule where the workload name is a subdomain of `defaultDomain`.

For example, with `defaultDomain` set to `example.com` in the plugin configuration, an enabled workload named `app` gets a
router and service named `app`, a rule of ``Host(`app.example.com`)``. 
If no usable guest IP is resolved and no explicit backend URL/IP label is set,
the generated backend server hostname also falls back to this domain, for
example `http://app.example.com:8080`.

#### Common Shorthand Labels

- `name`: override the default router/service name.
- `port`: backend HTTP port.
- `scheme`: backend scheme, usually `http` or `https`.
- `serverstransport`: HTTP servers transport reference.
- `insecureskipverify`: generate and attach an HTTP servers transport with `insecureSkipVerify=true`.
- `middlewares`: comma-separated HTTP router middleware references.
- `entrypoints`: comma-separated HTTP router entry points.

There are also shorthands for TCP and UDP services, some of which can be seen in the examples further down.

### Full Labels

The other method is using full Docker style labels like this:

```
traefik.enable=true
traefik.http.routers.traefik.rule=Host(`traefik.poppcorn.net`)
traefik.http.services.traefik.loadbalancer.server.port=8080

traefik.http.routers.traefik.middlewares=local-only@file,compress-all@file
```

These full labels can be anywhere in the VM/container notes, as long as they start at the beginning of a line.


## Plugin Configuration

The plugin currently supports these configuration options:

- `pollInterval`: how often to poll for changes in VM/container notes.
- `metadataMode`: label extraction mode, supported values are:
  - `fenced`: parse only ```` ```traefik ```` code blocks. This is the default.
  - `loose`: parse any `traefik.*` labels found in notes.
  - `auto`: use fenced blocks when present, otherwise fall back to loose labels.
- `defaultDomain`: the default domain to configure services and routers under.
- `pve`: Proxmox API and inventory discovery options.
  - `endpoint`: the Proxmox API endpoint URL.
  - `tokenID`: the Proxmox API token ID.
  - `token`: the Proxmox API token.
  - `timeout`: the Proxmox API HTTP request timeout.
  - `insecureSkipVerify`: whether to skip TLS certificate verification for the Proxmox API endpoint.
  - `skipStopped`: whether to skip stopped VMs/containers.
  - `skipIPResolution`: whether to skip automatic IP resolution for VMs/containers.
  - `ipMode`: which resolved guest IP versions to register. Supported values are `ipv4`, `ipv6`, and `ipv4/6`; the default is `ipv4`. This has no effect when `skipIPResolution` is true.
  - `defaultInterfaces`: default guest interface name patterns used for automatic IP resolution. The default is `eth*`, `enp*`, and `eno*`.
  - `maxConcurrency`: maximum concurrent per-guest PVE API calls during scans.
  - `nodes`: limits which PVE nodes are scanned.
  - `requiredTags`: a list of tags that must be present on VMs/containers to be included.

### IP Resolution

When `pve.skipIPResolution` is false, the provider resolves backend IPs before
building Traefik services.

- `pve.ipMode: ipv4` registers only IPv4 guest IPs. This is the default.
- `pve.ipMode: ipv6` registers only IPv6 guest IPs.
- `pve.ipMode: ipv4/6` registers both IPv4 and IPv6 guest IPs.
- `pve.defaultInterfaces` controls which guest interfaces are considered when
  no workload-specific `pve.interfaces` label is set. By default this is
  `eth*`, `enp*`, and `eno*`.

For VMs, IP discovery prefers QEMU guest-agent interface data and falls back to
static Cloud-Init `ipconfigN` entries from the VM config when present. For
containers, IP discovery uses the LXC interface data from Proxmox.

To limit automatic IP resolution to specific guest interfaces for one workload,
add a `pve.interfaces` label inside that workload's Traefik note block:

```traefik
enable=true
port=8080
pve.interfaces=eth*,ens18
```

The value is a comma-separated list of interface name patterns. `*` matches any
sequence and `?` matches one character, so `eth*` matches `eth0`, `eth1`, and
similar names. When `pve.interfaces` is set, it overrides
`pve.defaultInterfaces`, and only matching guest-agent/LXC interfaces are
considered. If no matching interface provides a usable IP, the service falls
back to the configured backend hostname behavior.

If no usable IP is found, backend servers fall back to
`<workload-name>.<defaultDomain>` when `defaultDomain` is configured, otherwise
to the legacy `<workload-name>.<node-name>` hostname. Explicit labels such as
`ip`, `http.services.<name>.loadbalancer.server.ip`, or
`http.services.<name>.loadbalancer.server.url` still override automatic
resolution.

## Development Notes

### Traefik / Yaegi Compatibility

Traefik loads provider plugins through the embedded Yaegi interpreter instead
of compiling them with the local Go toolchain. For that reason, runtime plugin
code must pass both normal Go tests and an interpreted import test.

The current Yaegi regression test has proven these constructs incompatible in
runtime-imported plugin code:

- `slices.Clone`, `slices.Sort`, and `slices.SortFunc`
- the `min` builtin
- `sync.WaitGroup.Go`
- `range` over integers
- unused self-referential label back-pointers such as `ProtocolSet -> *Set`
  and `Resource -> *Set`

Prefer simple `sort` helpers, explicit loops, explicit `WaitGroup` usage, and
small local helper functions where needed. Keep the label graph acyclic unless a
back-reference is actually required and covered by the Yaegi import test. Other
newer language or standard library conveniences should be checked with the Yaegi
import test before use:

```bash
go test -tags yaegi ./...
```


## Examples

Basic HTTP service:

````md
```traefik
enable=true
port=8080
middlewares=local-only@file,compress-all@file
```
````

HTTPS backend with skipped backend certificate verification:

````md
```traefik
enable=true
scheme=https
port=443
insecureskipverify=true
```
````

HTTPS backend with a file-provider servers transport:

````md
```traefik
enable=true
scheme=https
port=443
serverstransport=ignore-ssl@file
```
````

Provider-managed servers transport:

````md
```traefik
enable=true
port=8443
scheme=https
http.services.app.loadbalancer.serverstransport=ignore-ssl
http.serverstransports.ignore-ssl.insecureskipverify=true
http.serverstransports.ignore-ssl.forwardingtimeouts.dialtimeout=5s
```
````

TCP service:

````md
```traefik
enable=true
tcp.entrypoints=postgres
tcp.rule=HostSNI(`pg.example.com`)
tcp.port=5432
```
````

Per-node Traefik instance:

```yaml
providers:
  plugin:
    traefik-pve-provider:
      pve:
        endpoint: https://pve.example.com
        tokenID: root@pam!traefik
        token: your-token-secret
        nodes:
          - pve-1
        requiredTags:
          - traefik
```

## Proxmox Permissions

To use the PVE API, you need a dedicated token with read-only (recommended) privileges. A typical role is:

```bash
pveum role add traefik-provider -privs "VM.Audit,VM.Monitor,Sys.Audit,Datastore.Audit"
pveum user token add root@pam traefik
pveum acl modify / -token 'root@pam!traefik' -role traefik-provider
```

On Proxmox VE versions where guest-agent access is split out, include
`VM.GuestAgent.Audit` so VM IP discovery can read guest-agent network
interfaces. If `pve.skipIPResolution: true` is set, guest-agent and LXC interface
permissions are not needed, but Traefik will need to reach the hostname
fallbacks this provider generates.
