# Wormhole Pauser (SVM)

Anchor implementation of the delegated pauser described in
[whitepapers/0018_pauser.md](../../whitepapers/0018_pauser.md). An immutable
program that lets a Wormhole-governed signer set propose and execute arbitrary
`pause()`-style CPIs against downstream protocols (WTT, NTT, etc.) without
requiring a fresh super-majority guardian governance message for every action.

## Programs

| Program           | Path                        | Purpose                                              |
| ----------------- | --------------------------- | ---------------------------------------------------- |
| `wormhole_pauser` | `programs/wormhole-pauser/` | The pauser itself: config, propose, approve, cancel. |
| `mock_pausable`   | `programs/mock-pausable/`   | Test-only target used by the integration tests.      |

The pauser issues every downstream call via `invoke_signed` with the
`[b"authority"]` PDA, so configuring a downstream protocol's `pauser` role to
this PDA is what binds the two together.

## How It Works

```text
                  ┌──────────┐
         propose  │          │  threshold reached
         ────────▶│ Pending  │──────────────────▶ Executed
                  │          │
                  └────┬─────┘
                       │ block.timestamp >= expiresAt
                       └─────────────────────────────▶ Expired
```

1. **`submit_config`** — applies a `DelegatedPauser` `SetConfigSolana` (action
   `2`) governance VAA. The action must match the platform, the chain ID must
   match this chain (Solana = `1`), and the message index must be exactly
   `currentIndex + 1`. The VAA digest is bound to a `consumed_vaa` PDA so the
   same VAA cannot be replayed. Signers, threshold, and expiry duration all
   live in the singleton `Config` PDA (`[b"config"]`).
2. **`propose`** — a current signer creates a `Proposal` PDA
   (`[b"proposal", id_le]`) carrying `(target_program, account_metas, data)`.
   The proposer is auto-approved, so a `threshold == 1` config executes the
   CPI atomically in the same transaction. Otherwise the proposal sits at
   `approval_count = 1` until further approvals.
3. **`approve`** — another current signer approves; on the threshold-meeting
   approval the program flips `executed = true` and `invoke_signed`s into the
   stored target. If the CPI reverts, the entire transaction reverts (effect
   first, then interaction), so signers can retry without permanently burning
   their approval.
4. **`cancel_approval`** — a signer that has approved an active proposal can
   withdraw their approval.

A `SetConfig` rotates the on-chain config index and **implicitly invalidates
every prior proposal** via `proposal.config_index == config.config_index`.

## Build

The package follows the same Anchor 0.31 / Solana 2.1.20 layout as
`svm/delegated-manager-set/`. From this directory:

```sh
yarn install
anchor build
```

The first build under a fresh checkout will create program keypairs under
`target/deploy/`. Reconcile their pubkeys with the placeholder IDs in
`Anchor.toml` and `declare_id!` (or grind a vanity keypair) before running the
test validator.

## Test

```sh
anchor test
```

`Anchor.toml` boots a local validator with the testnet Wormhole core bridge and
the verify-vaa shim pre-loaded from `tests/artifacts/`, plus a guardian set
fixture (`tests/accounts/core_bridge_testnet/guardian_set_0.json`) whose key
matches the dev guardian (`GUARDIAN_KEYS[0]` from the Wormhole SDK). The TS
tests use `MockGuardians` to sign `SetConfigSolana` VAAs against that fixture
and `MockPausable` to assert the threshold-meeting CPI actually fires.

## Submitting a Governance VAA on Mainnet/Devnet

`anchor build` exposes the program IDL under `target/idl/wormhole_pauser.json`.
Use it with the Anchor TS client to call `submit_config` against a guardian-
signed `DelegatedPauser` VAA. The required PDAs are:

- `consumed_vaa`: `[b"consumed_vaa", keccak256(keccak256(body))]`
- `config`: `[b"config"]`
- `authority`: `[b"authority"]`
- `proposal`: `[b"proposal", proposal_id_u64_le]`

See `tests/wormhole-pauser.ts` for an end-to-end example, including how to
post the guardian signatures via the verify-vaa shim before submitting.
