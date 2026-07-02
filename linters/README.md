# Golang CI Lints

Custom Go linters used on Wormhole CI. Each linter is a [golangci-lint
module plugin](https://golangci-lint.run/plugins/module-plugins/) and lives
as its own Go module under `rules/<linter>/`. 
  
Prefer to use the `release` builds.

Currently supported linters:

| Name           | Purpose                                                  | Features                                                                                                                  |
| -------------- | -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `channelcheck` | Flag channel usage patterns that can deadlock or block.  | • Blocking channel sends outside a `select`<br>• Unbuffered channel creation (`make(chan T)`)<br>• Empty `default:` that silently drops a send (advisory, off by default)<br>• Ignore specific channels |


## Build & test
```
make build                 # bin/wormhole-lint
make build-golangci-lint   # bin/wormhole-golangci-lint
make test                  # root + each rules/<linter> module
```

`make build-golangci-lint` first installs the pinned upstream
`golangci-lint` into `bin/` (per `GOLANGCI_LINT_HASH`), then runs
`golangci-lint custom`. 

## Use

Standalone:

```
bin/wormhole-lint ./...
```

Via the custom golangci-lint:

```
bin/wormhole-golangci-lint run --timeout=10m ./...
```

## Enable a plugin in `.golangci.yml`

Module plugins are addressed under `linters.settings.custom.<name>` and
enabled in `linters.enable` by the plugin's registered name. Example for
`channelcheck`:

```yaml
version: "2"
linters:
  enable:
    - channelcheck
  settings:
    custom:
      channelcheck:
        type: module
        description: reports channel blocking issues
        settings:
          blocking: true
          unbuffered: false
          bufferMax: 0
          emptyDefault: false
          ignoreChannelsByName: [errC]
```

`channelcheck` settings:

| Setting                | Type       | Default | Description                                                                                                                              |
| ---------------------- | ---------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `blocking`             | `bool`     | `true`  | Flag blocking sends that have no escape (timer, `default:`, or `ctx.Done()`), including sends outside any `select`.                       |
| `unbuffered`           | `bool`     | `false` | Flag unbuffered channel creation (`make(chan T)` / `make(chan T, 0)`).                                                                    |
| `bufferMax`            | `uint`     | `0`     | Flag channel buffers larger than this. `0` disables the check.                                                                            |
| `emptyDefault`         | `bool`     | `false` | Advisory: flag an empty `default:` in a `select` with a send, since it silently drops the send. Off by default — it fires on the idiomatic non-blocking/coalescing-send pattern, so enable it only where dropping should always be logged or documented. |
| `ignoreChannelsByName` | `[]string` | `[]`    | Channel/field names whose direct sends are exempt from the blocking-send, empty-default, and `ctx.Done()` checks.                        |

## Development

### Adding a new linter

1. Scaffold the module:
   ```
   mkdir -p rules/<linter> && cd rules/<linter>
   go mod init github.com/wormhole-foundation/wormhole/linters/rules/<linter>
   ```
2. Implement `<linter>.go` following the channelcheck reference
   (`rules/channelcheck/channelcheck.go`):
   - Export an `Analyzer` of type `*analysis.Analyzer`.
   - Define a `Settings` struct and a `New(any) (register.LinterPlugin, error)`
     constructor that decodes settings via
     `register.DecodeSettings[Settings]`.
   - Implement `BuildAnalyzers()` and `GetLoadMode()` on your plugin type.
   - In `init()`, call `register.Plugin("<name>", New)` so
     `golangci-lint custom` picks it up.
3. Add tests + fixtures under `rules/<linter>/testdata/` following
   `rules/channelcheck/channelcheck_test.go`.
4. Wire it into the root module so the aggregator can import it:
   - In root `go.mod`, add
     `require github.com/wormhole-foundation/wormhole/linters/rules/<linter> v0.0.0` and
     `replace github.com/wormhole-foundation/wormhole/linters/rules/<linter> => ./rules/<linter>`.
   - In `cmd/wormhole-lint/main.go`, add the import and append
     `<linter>.Analyzer` to the `multichecker.Main` call.
5. Wire it into `.custom-gcl.yml` so the custom golangci-lint picks it up:
   ```yaml
   plugins:
     - module: github.com/wormhole-foundation/wormhole/linters/rules/<linter>
       path: ./rules/<linter>
   ```
6. `make test && make build && make build-golangci-lint` to verify.
7. Add linter to `.golangci.yml`.
8. Fix linter errors in the monorepo with legitimate changes or a `nolint` comment.

### Layout

```
.custom-gcl.yml              # plugin manifest for `golangci-lint custom`
Makefile
go.mod                       # root module: cmd/* + replaces for each rules/<linter>
cmd/
  wormhole-lint/             # multichecker aggregator binary
rules/
  channelcheck/              # standalone Go module per linter
    channelcheck.go
    channelcheck_test.go
    go.mod
    testdata/
```
