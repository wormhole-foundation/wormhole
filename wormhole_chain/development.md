# Develop

## Scaffolding stuff

TODO: expand explanation here

```shell
starport scaffold type guardian-key key:string --module wormhole --no-message
```

modify `proto/wormhole/guardian_key.proto` (string -> bytes)

```shell
starport generate proto-go
```
