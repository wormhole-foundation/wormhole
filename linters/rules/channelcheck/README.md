## Channel Check 

Channels are a great feature of Golang but have several footguns that can lead to deadlocks. In particular, if the receiving channel stops processing the messages, a *non-blocking* channel send would fail to continue. In certain mission-critical sections of code, this could lead to a complete deadlock. 
  
This linter currently has three features: 
- Identify blocking sends
- Identify non-buffered channel creation
- Identify buffered channel size exceeds maximum size checks 
  
Many of these will lead to false positives or situations where we *want* a blocking channel send. In these cases, `nolint:channelcheck` is easy to add (assuming this is integrated directly with golangci-lint). Regardless, having this issue pointed out automatically is a good way to fix bugs.

## Configuration

Each option can be set two ways: as a golangci-lint module setting (under
`settings.custom.channelcheck.settings:`, using the **Setting** name) or as a
standalone analyzer flag (when running the `wormhole-lint` binary, using the
**Flag** name).

| Setting                   | Flag         | Type       | Default | Description                                                                                     |
| ------------------------- | ------------ | ---------- | ------- | ----------------------------------------------------------------------------------------------- |
| `CheckBlockingSends`      | `blocking`   | bool       | `true`  | Flag blocking sends that lack a `default`/timeout/ticker escape in their enclosing `select`.    |
| `CheckUnbufferedChannels` | `unbuffered` | bool       | `false` | Flag creation of unbuffered channels (`make(chan T)`).                                           |
| `CheckBufferAmount`       | `bufferMax`  | uint64     | `0`     | Flag buffered channels whose size exceeds this max. `0` disables the check.                      |
| `IgnoreChannelsByName`    | *(none)*     | []string   | `[]`    | Channel/field names whose direct sends are exempt from the blocking-send check (e.g. `errC`). Settings-only; no standalone flag. |