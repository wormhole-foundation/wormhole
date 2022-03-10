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

```shell
starport scaffold message register-account-as-guardian guardian-pubkey:GuardianKey address-bech32:string signature:string --desc "Register a guardian public key with a wormhole chain address." --module wormhole --signer signer
```
