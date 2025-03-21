# Shim Interfaces and Examples

For those integrators that use Anchor and IDLs, this directory defines Anchor
interfaces for the Wormhole Shim programs and example programs that demonstrate
how to use these interfaces.

## Tests

To perform the Anchor Typescript tests, run:

```sh
make test
```

Shim programs are built from the parent directory and moved to this directory
for the Anchor test. Interfaces are built to generate IDLs so the tests can
build instructions to interact with these programs in a local validator.

The e2e tests can then be run with `npx tsx tests/e2e.ts` against a running Tilt
environment with at least `tilt up -- --solana --manual`.

These are _not_ currently run within Tilt. It would be prudent to both build and
run these within Tilt once the shim approach has been finalized.
