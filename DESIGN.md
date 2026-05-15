# Design Notes

## Traefik Label Parsing

The provider accepts note entries that look like Traefik labels, but Traefik does
not parse those labels for plugin providers. The provider must translate note
configuration into `dynamic.Configuration` itself.

The label path is implemented as a small lexer, recursive-descent parser,
syntax AST, and interpreter. AST node types live in `traefik/ast`;
tokenization lives in `traefik/ast/lexer`; parse context, parse errors, and
recursive-descent parsing live in `traefik/ast/parser`. The `traefik/labels`
package owns parsed label sets, diagnostics, resource indexes, and typed
lookups. The parent `traefik` package consumes that AST-backed representation
when building provider configuration.

### Pipeline

```text
PVE notes
  -> metadata parser: fenced/loose key=value labels
  -> Traefik label lexer: typed tokens
  -> recursive-descent parser: assignment AST
  -> interpreter: Traefik dynamic configuration
```

### Tokens

The lexer treats known structural words as dedicated token types. Examples:

```text
http.services.app.loadbalancer.server.port=8080
```

becomes conceptually:

```text
HTTP Dot Services Dot Identifier("app") Dot LoadBalancer Dot Server Dot Port Equals Number(8080)
```

Unknown object names are identifiers. Known grammar words such as `http`,
`services`, `loadbalancer`, `server`, and `port` are token kinds.

### AST

The parser produces a true syntax tree: an assignment whose target is a linked
chain of token-backed property nodes, plus a typed value node. The target path
keeps structural context, so `server.port` and `healthcheck.port` can use the
same `Port` token without requiring global semantic scalar kinds:

```text
Assignment
  Target
    HTTP
      Services
        Identifier("app")
          LoadBalancer
            Server
              Port
  Value
    Number(8080)
```

The interpreter decides meaning from parent context. The AST does not encode
fields as semantic leaf enums.

### Shorthand Expansion

Shorthand labels are parsed into the same AST shape as their full equivalents.
There should be no `<default>` placeholder in the final AST. The parser receives
a parse context with the default object name and expands directly.

Examples:

```text
port=8080
```

parses as:

```text
HTTP -> Services -> Name(defaultName) -> LoadBalancer -> Server -> Port(8080)
```

```text
tcp.port=5432
```

parses as:

```text
TCP -> Services -> Name(defaultName) -> LoadBalancer -> Server -> Port(5432)
```

### Interpreter

Avoid a visitor pattern for now. We only need one main operation:
interpreting parsed assignments into Traefik configuration. `labels.Set`
stores the parsed assignments plus lightweight indexes of discovered routers,
services, and transports. Builder code asks those objects for typed values by
token path, so the raw label strings are parsed once and not reinterpreted in
the config builders.
