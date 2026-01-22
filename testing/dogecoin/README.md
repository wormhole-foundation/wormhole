# Dogecoin Multisig Bridge Test

This folder contains a script to test deposits and withdrawals via the Wormhole testnet guardian's 5-of-7 multisig Dogecoin Delegated Manager Set.

## Testing Artifacts

https://doge-testnet-explorer.qed.me/tx/8d55cbffc83f27ec16fb3da9c550857533169233c6683f8d97986b14b928de81
https://doge-testnet-explorer.qed.me/tx/2916f3063844f0e2721f81041c3dc2ac02790d24ce3a1b31b8c1fb2e908d52c0

## Prerequisites

### Bun

```bash
curl -fsSL https://bun.sh/install | bash
```

### Install Dependencies

```bash
bun ci
```

## Dogecoin Testnet Wallet (dogecoin-keygen.ts)

Generates a Dogecoin testnet keypair with WIF-encoded private key and P2PKH address.

```bash
bun run dogecoin-keygen.ts
```

## Solana Devnet Wallet (solana-keygen.ts)

Generates a Solana devnet keypair.

```bash
bun run solana-keygen.ts
```

Note: As the Delegated Manager Set feature is currently permissioned, in order for this test to work, your Solana devnet address has to be in the allowlist of the running testnet guardian.

## Testnet Deposit Script (deposit-testnet.ts)

Deposits DOGE to a P2SH multisig address controlled by the delegated manager set. The script builds a custom redeem script with embedded metadata (emitter chain, emitter contract, sub-address seed) and creates a 5-of-7 multisig.

### Prerequisites

- `dogecoin-testnet-keypair.json` - Funded Dogecoin wallet (from `dogecoin-keygen.ts`)
- `solana-devnet-keypair.json` - Solana keypair for emitter contract address (from `solana-keygen.ts`)

### Usage

```bash
bun run deposit-testnet.ts
```

## Testnet Withdraw Script (withdraw-testnet.ts)

Withdraws DOGE from the P2SH multisig address back to the original sender. Collects 5-of-7 signatures from the delegated manager set to satisfy the multisig requirement.

### Prerequisites

- `solana-devnet-keypair.json` - Same Solana keypair used for deposit
- `deposit-info.json` - Created by running `deposit.ts`

### Usage

```bash
bun run withdraw-testnet.ts
```
