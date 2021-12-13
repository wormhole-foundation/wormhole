# Contributing

Wormhole is an open-source project licensed under the permissive Apache 2 license. Contributions are greatly
appreciated and will be reviewed swiftly.

Wormhole is a mission-critical, high-stakes project. We optimize for quality over quantity. Design processes
and code reviews are our most important tools to accomplish that.

- All new features must first be discussed in a GitHub issue before starting to implement them. For
  complex features, it can be useful to submit a [formal design document](design/template.md).

- Development happens on a long-lived development branch (`dev.v2` and `dev.v1`).
  Every change going into a development branch is reviewed individually (see below). Release branches may be used 
  to support in-the-wild releases of Wormhole. We aim to support at most two release
  branches at the same time. Changes can be cherry-picked from the development branch to release branches, but
  never from release branches to a development branch.
  
- Releases are first tested on a testnet.

- Commits should be small and have a meaningful commit message. One commit should, roughly, be "one idea" and
  be as atomic as possible. A feature can consist of many such commits.
  
- Feature flags and interface evolution are better than breaking changes and long-lived feature branches.
  
- We optimize for reading, not for writing - over its lifetime, code is read much more often than written.
  Small commits, meaningful commit messages and useful comments make it easier to review code and improve the
  quality of code review as well as review turnaround times. It's much easier to spot mistakes in small,
  well-defined changes.

Documentation for the in-the-wild deployments lives in the
[wormhole-networks](https://github.com/certusone/wormhole-networks) repository.

See [DEVELOP.md](./DEVELOP.md) for more information on how to run the development environment.

## Contributions FAQ

### Can you add \<random blockchain\>?

The answer is... maybe? The following things are needed in order to fully support a chain in Wormhole:

- The Wormhole mainnet is governed by a DAO. Wormhole's design is symmetric - every guardian node needs to run
  a node or light client for every chain supported by Wormhole. This adds up, and the barrier to support new
  chains is pretty high. Your proposal should clearly outline the value proposition of supporting the new chain.
  **Convincing the DAO to run nodes for your chain is the first step in supporting a new chain.**
  
- The chain needs to support smart contracts capable of verifying 19 individual secp256k1 signatures.

- The smart contract needs to be built and audited. In some cases, existing contracts can be used, like with
  EVM-compatible chains.
  
- Support for observing the chain needs to be added to guardiand.

- Web wallet integration needs to be built to actually interact with Wormhole.

The hard parts are (1) convincing the DAO to run the nodes, and (2) convincing the core development team to
either build the integration, or work with an external team to build it.

Please do not open a GitHub issue about new networks - this repository is only a reference implementation for
Wormhole, just like go-ethereum is a reference implementation for Ethereum. Instead, reach out to the
[Wormhole Network](https://wormholenetwork.com).

### Do you support \<random blockchain innovation\>?

Probably :-). At its core, Wormhole is a generic attestation mechanism and is not tied to any particular kind
of communication (like transfers). It is likely that you can use the existing Wormhole contracts to build your
own features on top of, without requiring any changes in Wormhole itself.

Please open a GitHub issue outlining your use case, and we can help you build it!
