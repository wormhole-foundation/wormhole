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

## Iterating on the chain quickly

First initialise the chain using starport:
```shell
starport chain init --home build
```

This step only needs to be repeated when you want to reset the blockchain state,
or when config.yml is changed.

Then update `build/config/client.toml` so that the `chain-id` field reads
`"wormholechain"`. This has to be done after each `starport chain init`.

Then run the blockchain:
```shell
cd cmd/wormhole-chaind
go run main.go start --home ../../build
```

Now each time you make a change to the blockchain code, you can just kill this
process and rerun the command.

You can interact with the blockchain by using the same go binary:
```shell
cd cmd/wormhole-chaind
go run main.go tx tokenbridge execute-governance-vaa 01000000000100e86068bfd49c7209f259110dc061012ca6d65318f3879325528c57cf3e4950ff1295dbde77a4c72f3aee29a32a07099257521674725be8eb8bbd801349a828c30100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000038502e100000000000000000000000000000000000000000000546f6b656e4272696467650100000001c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f --from tiltGuardian --home ../../build
```

Note the flags `--from tiltGuardian --home ../../build`. These have to be passed
in each time you make a transaction (the `tiltGuardian` account is created in
`config.yml`). Queries don't need the `--from` flag.

At this stage, it might be even faster to compile the binary and invoke it directly:
```shell
cd cmd/wormhole-chaind
go build main.go
# then
./main tx tokenbridge ...
```
