# Threat model

This file steers the Anthropic OSS Scanner (https://github.com/anthropics/oss-scanner). Read it together with:

- `SECURITY_CONTEXT.md`: design decisions, accepted risks and bug classes that are **not** findings. Anything
  matching an entry there is closed as informational, so check it before reporting.
- `SAFETY_CRITICAL_CODE.md`: the coding standard. Deviations from it are hints about where bugs hide, but a
  style deviation on its own is not a finding.
- `SECURITY.md`: the security policy, including the Immunefi bug bounty program.
- `docs/`: protocol documentation, including the VAA and governance message formats.

## What this project does and where untrusted input enters

Wormhole is a cross-chain messaging protocol. Contracts ("core bridges") on each connected chain emit messages.
A set of independent Guardian nodes observes those messages, and when a quorum (13 of 19) has signed an
observation, the signatures form a VAA (Verifiable Action Approval). Contracts on any other chain verify
the VAA signatures against the current guardian set and act on the payload. The Token Bridge and NFT Bridge
are applications built on top: they lock or burn assets on one chain and mint or release them on another.
Wormhole secures billions of dollars of assets; a single bug can be catastrophic.

The security model is: **nothing except a quorum of guardian signatures authorizes a message.** Everything
that flows into the system is attacker controlled:

- **Chain data read by guardian watchers** (`node/pkg/watchers/*`): blocks, logs, receipts, events and
  transaction data from RPC nodes. A watcher must only ever emit an observation for a message that was
  really published by the real core bridge on a finalized block. Anyone can call the core bridges and
  create arbitrary messages, so emitter address, sequence, consistency level and payload are all untrusted.
- **Peer-to-peer traffic between guardians** (`node/pkg/p2p`, `node/pkg/processor`): observations,
  signed VAAs, heartbeats, governor status, and observation requests. Any peer can be malicious.
- **VAAs** (`sdk/vaa`, and every on-chain VAA parser): any bytes anyone submits to a contract or to the
  node's APIs. Parsing and signature verification are the most security critical code in the repository.
- **Governance VAAs**: guardian set updates, contract upgrades, fee and transfer-limit changes. They are
  signed by the guardians, but each consumer must still check the governance module, action, target chain
  and emitter, and must prevent replay.
- **Messages and tokens submitted to the Token/NFT bridges**: attacker-chosen token contracts, amounts,
  decimals, recipients, and payloads. Registering a token is permissionless by design.
- **Public RPC / API surface of the node** (`node/pkg/publicrpc`, `node/pkg/adminrpc`, `node/pkg/server`):
  the admin API is meant to be local-only; the public gRPC/REST APIs are exposed to the Internet.
- **Guardian operator inputs**: configuration flags and admin commands are trusted.

## Scope of this scan

This image contains only the guardian node (`node/`) and the Go SDK (`sdk/`), with `wormchain/` present
because the node module depends on it. The on-chain contracts in this repository (`ethereum/`, `solana/`,
`cosmwasm/`, `algorand/`, `aptos/`, `near/`, `sui/`, `terra/`, `svm/`) are not built, not tested and not in
scope for this scanner, although they remain in scope for the bug bounty program. Read them only when needed
to understand what a watcher is parsing.

## Components that matter most / least

Most important, in order:

1. `sdk/vaa`: VAA parsing, signature verification, quorum calculation. Shared by the node and the Go SDK.
2. `node/pkg/processor`: observation aggregation, quorum and VAA creation.
3. `node/pkg/watchers/*`: per-chain message observation, finality handling, reobservation.
4. `node/pkg/p2p`, `node/pkg/node`, `node/pkg/common`: the gossip layer, message validation, bounds.
5. `node/pkg/governor`, `node/pkg/notary`, `node/pkg/accountant`, `node/pkg/txverifier`: defence-in-depth
   rate limiting and transfer verification. See `SECURITY_CONTEXT.md` for what is accepted behaviour here.
6. `wormchain`: the Cosmos SDK based Gateway chain, in particular `x/wormhole` (guardian set and governance
   handling, `MsgExecuteGovernanceVAA`, `MsgStoreCode`, `MsgInstantiateContract`), `x/ibc-composability-mw`
   and the IBC hooks. Lower priority than the node: it is a build dependency of the guardian and only
   partly exercised by its tests.

Less important, still in scope: `node/pkg/publicrpc`, `node/pkg/adminrpc`, `node/pkg/server` (API input
handling and resource bounds), `node/cmd` (flag handling), `sdk/` other than `sdk/vaa`.

Out of scope, do not report:

- `devnet/`, `testing/`, `Tiltfile`, `tilt_modules/`, `scripts/`, `tools/`, `linters/`, `codeql/`,
  `lp_ui/`, `dashboards/`, `third_party/`, every `Dockerfile`, every `*_test.go`.
  Local-development and CI code only.
- `deployments/` and `audits/`: data, not code.
- `sdk/js` (deprecated, replaced by the separate TypeScript SDK repository).
- Every on-chain contract directory, as listed under "Scope of this scan".
- Third-party code in the Go module cache.

## How to exercise it

The image has Go 1.25 with every module already fetched and every package and test binary already compiled
(with and without `-race`). Everything below works without a network. The scanner's agent machine is small
(2 CPUs, 8 GB), so run the package or test you care about rather than whole trees where you can.

- `cd node && go test ./pkg/processor/...`, `cd sdk && go test ./vaa/...`, or `make test-fast` at the root
  for everything with `-short`. The full suite with the race detector is `make test-coverage` and takes a
  long time on 2 CPUs.
- `build/bin/guardiand` is the built node; `guardiand --help` lists its subcommands, and `guardiand admin`
  holds the admin tools (for example `guardiand admin governance-vaa-verify`).
- Fuzz targets exist, for example `cd sdk/vaa && go test -fuzz FuzzCalculateQuorum -run FuzzCalculateQuorum`.
- Tests tagged `integration` need a running devnet and cannot run here; `make test-integration-compile`
  only compiles them.
- wormchain: `cd wormchain && go test ./x/wormhole/...`. Its test binaries are not precompiled, so the
  first run is slower.
- `node/pkg/manager` has tests that call a public Sepolia RPC endpoint; they fail offline and that is not a
  finding. Everything else in `make test-fast` passes without a network.

## How you rate severity

Rate findings as the Immunefi bug bounty program would (see `SECURITY.md`):

- **Critical**: creating, accepting or executing a VAA (including a governance VAA) that a quorum of the
  current guardian set did not sign; minting, unlocking or stealing bridged assets; replaying a VAA on a
  chain that has already consumed it; a guardian emitting an observation for a message the core bridge did
  not publish, or publishing it from a non-finalized or reorged block; bypassing guardian set expiry; a
  watcher or processor bug that lets one chain's message be attributed to another emitter or chain.
- **High**: anything that lets a remote peer or RPC response crash, hang or permanently wedge a guardian
  (a guardian that stops signing reduces the network below quorum); unbounded memory or disk growth driven
  by untrusted input; bypassing the governor, notary, transfer verifier or accountant with a single
  transaction (rather than by sending real value through them); governance VAA handling bugs in the node
  or in wormchain.
- **Medium**: denial of service that recovers on its own or with an operator restart; bugs that need an
  unusual but reachable configuration; incorrect results in the public RPC API.
- **Low / informational**: findings that need a malicious guardian operator, a compromised RPC node that
  the guardian is configured to trust, or local access to the admin socket; gas inefficiencies; anything
  under "Non-issues" in `SECURITY_CONTEXT.md`.

A finding that assumes control of a quorum of guardian keys, or an attacker-controlled token that only harms
its own users, is not a finding at all; see `SECURITY_CONTEXT.md`.

## Anything to leave alone

- Everything listed under "Non-issues" in `SECURITY_CONTEXT.md`.
- Volumetric denial of service (many messages or many peers) against the p2p layer. libp2p limits message
  sizes to 1 MB and the per-peer rate limits are intentional.
- Float truncation and price imprecision in the governor.
- Fee logic in the token bridges: fees are zero in production.
- Governance VAAs being accepted out of order; this is deliberate.
- Known-vulnerable version reports for pinned dependencies without a reachable path from untrusted input.
