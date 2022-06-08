# Liquidity Provider UI

This UI can be used for creating and managing migration pools as defined by `solana/migration` and `ethereum/contracts/bridge/utils/Migrator.sol`

## Quick Start

```bash
npm ci
REACT_APP_CLUSTER=mainnet npm start
```

Navigate to http://localhost:3000/

## Create a new Ethereum pool

> Please ensure your wallet is connected to the desired chain! These instructions are suitable for any EVM chain.

1. Click "Create a New Ethereum Pool".
1. Click "Connect" to connect your Metamask.
1. Paste the address of the token you want users to migrate _from_ in the "From Token" field. This is the 'old' token that users currently hold.
1. Paste the address of the token you want users to migrate _to_ in the "To Token" field. This is the 'new' token that users will receive when migrating.
1. Click "Create".
1. Confirm in your wallet.
1. After the transaction is successful, **be sure to note the address of the migration pool**. This is required in order to manage the pool.

In order for users to be able to use this newly created pool, "To" tokens must be added to the pool.

## Add liquidity to an existing Ethereum pool

> Please ensure your wallet is connected to the desired chain! These instructions are suitable for any EVM chain.

1. Click "Interact with an existing Ethereum Pool".
1. Click "Connect" to connect your Metamask.
1. Paste the address you received when creating the pool in the "Migrator Address" field.
1. You should see the Pool Balances load.
1. Under "Add Liquidity", type an amount in the "Amount to add" field.
1. Click "Add Liquidity".
1. Confirm in your wallet.
1. After the transaction is successful, the Pool Balances and Connected Wallet Balances should refresh.

## Create and manage Solana pools

1. Click "Manage Solana Liquidity pools".
1. Connect your wallet.
1. Enter the From token and To token.

If a pool for those tokens has not been created, a "This pool has not been instantiated! Click here to create it." button will appear. Otherwise, the button will be disabled and read "This pool is instantiated."

Before you can add liquidity, you must create a Share SPL Token Account. Similarly, before redeeming shares, you must create a 'From' SPL Token Account.

## Prerequisites

NodeJS v14+

## Install

```bash
npm ci
```

## Develop

```bash
npm start
```

## Build for local tilt network

```bash
npm run build
```

## Build for testnet

```bash
REACT_APP_CLUSTER=testnet npm run build
```

## Build for mainnet

```bash
REACT_APP_CLUSTER=mainnet npm run build
```

## Test Server

```bash
npx serve -s build
```
