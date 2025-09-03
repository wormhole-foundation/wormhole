# Aztec Testnet Deployment Guide

This guide walks through deploying the Wormhole contract and Token contract on the Aztec testnet.

## Prerequisites

- Aztec CLI tools installed
- Access to Aztec testnet
- Basic understanding of Aztec wallet operations

## Alternative: Automated Deployment Script

For a fully automated deployment experience, you can use the `deploy.sh` script instead of following this manual guide:

```bash
# Make the script executable
chmod +x deploy.sh

# Run the automated deployment wizard
./deploy.sh
```

The `deploy.sh` script provides:
- Interactive configuration wizard
- Automatic error handling and retry logic
- Built-in dependency checking
- Automatic contract address extraction and updates
- Comprehensive logging and status reporting

## Manual Deployment Steps

If you prefer to deploy manually or need more control over the process, follow the steps below:

## Environment Setup

### 1. Set Environment Variables

```bash
# Testnet configuration
export NODE_URL=https://aztec-alpha-testnet-fullnode.zkv.xyz
export SPONSORED_FPC_ADDRESS=0x19b5539ca1b104d4c3705de94e4555c9630def411f025e023a13189d0c56f8f22

# Owner private key 
export OWNER_SK=0x9015e46f2e11a7784351ed72fc440d54d06a4a61c88b124f59892b27f9b91301
```

## Wallet Creation

### 2. Create Wallets

#### 2a. Create Owner Wallet

```bash
aztec-wallet create-account \
    -sk $OWNER_SK \
    --register-only \
    --node-url $NODE_URL \
    --alias owner-wallet
```

**Note**: The owner wallet will be used to deploy the contract. You may want to record this address for your own reference, though it's not required for the deployment process.

#### 2b. Create Receiver Wallet

```bash
aztec-wallet create-account \
    --register-only \
    --node-url $NODE_URL \
    --alias receiver-wallet
```

**Important**: The receiver wallet is used as a **fee collector** for Wormhole messages. This wallet will receive all fees paid when publishing cross-chain messages.

#### 2c. Set Receiver Address Variable and Update addresses.json

```bash
# Extract the receiver address from the wallet creation logs
export RECEIVER_ADDRESS=<receiver_address_from_step_2b>

# Update the addresses.json file with the receiver address
jq --arg receiver "$RECEIVER_ADDRESS" '.receiver = $receiver' addresses.json > addresses.json.tmp && mv addresses.json.tmp addresses.json
```

## FPC Registration

### 3. Register Wallets with FPC

#### 3a. Register Owner Wallet

```bash
aztec-wallet register-contract \
    --node-url $NODE_URL \
    --from owner-wallet \
    --alias sponsoredfpc \
    $SPONSORED_FPC_ADDRESS SponsoredFPC \
    --salt 0
```

#### 3b. Register Receiver Wallet

```bash
aztec-wallet register-contract \
    --node-url $NODE_URL \
    --from receiver-wallet \
    --alias sponsoredfpc \
    $SPONSORED_FPC_ADDRESS SponsoredFPC \
    --salt 0
```

## Account Deployment

### 4. Deploy Accounts

> **Note**: You may encounter `Timeout awaiting isMined` errors, but this is normal. Continue with the next step.

#### 4a. Deploy Owner Wallet

```bash
aztec-wallet deploy-account \
    --node-url $NODE_URL \
    --from owner-wallet \
    --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc
```

#### 4b. Deploy Receiver Wallet

```bash
aztec-wallet deploy-account \
    --node-url $NODE_URL \
    --from receiver-wallet \
    --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc
```

## Contract Deployment

### 5. Deploy Token Contract

```bash
aztec-wallet deploy \
    --node-url $NODE_URL \
    --from accounts:owner-wallet \
    --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc \
    --alias token \
    TokenContract \
    --args accounts:owner-wallet WormToken WORM 18 --no-wait
```

**Important**: Wait for the transaction to be mined and capture the token contract address from the logs. You can check the transaction status at [AztecScan](http://aztecscan.xyz/).

### 6. Set Token Contract Address and Update addresses.json

#### 6a. Set Token Contract Address

```bash
# Set the token contract address (extract from step 5 deployment logs)
export TOKEN_CONTRACT_ADDRESS=<token_contract_address_from_step_5>
```

#### 6b. Update addresses.json

```bash
# Update the addresses.json file with the new token address
jq --arg token "$TOKEN_CONTRACT_ADDRESS" '.token = $token' addresses.json > addresses.json.tmp && mv addresses.json.tmp addresses.json
```

### 7. Mint Initial Tokens

#### 7a. Mint Private Tokens

```bash
aztec-wallet send mint_to_private \
    --node-url $NODE_URL \
    --from accounts:owner-wallet \
    --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc \
    --contract-address $TOKEN_CONTRACT_ADDRESS \
    --args accounts:owner-wallet accounts:owner-wallet 10000
```

#### 7b. Mint Public Tokens

```bash
aztec-wallet send mint_to_public \
    --node-url $NODE_URL \
    --from accounts:owner-wallet \
    --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc \
    --contract-address $TOKEN_CONTRACT_ADDRESS \
    --args accounts:owner-wallet 10000
```

### 8. Compile the Wormhole Contract

**Important**: The Wormhole contract must be compiled before deployment. If you've made any changes to the contract source code, compile it first:

**Note**: The contract contains hardcoded addresses in the `publish_message_in_private` function because private functions cannot access public storage. These addresses are automatically updated by the deployment script to match your deployed token and receiver addresses. If you're deploying manually, you may need to update these addresses in `src/main.nr` before compilation.

**Note**: This limitation is expected to be resolved in a future Noir language update, which will eliminate the need for hardcoded addresses.

**IMPORTANT**: There are 2 places where hardcoded addresses must be updated:
1. In the `publish_message_in_private` function (at the top of `src/main.nr`)
2. In the `test_verify_vaa_logic` test function (in the same file)

Both locations must use the same addresses to ensure consistency between the actual contract logic and the test functions.

```bash
# Compile the contract
aztec-nargo compile
```

### 9. Deploy Wormhole Contract

```bash
aztec-wallet deploy \
    --node-url $NODE_URL \
    --from accounts:owner-wallet \
    --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc \
    --alias wormhole \
    target/wormhole_contracts-Wormhole.json \
    --args 56 56 $RECEIVER_ADDRESS $TOKEN_CONTRACT_ADDRESS --no-wait --init init
```

**Note**: The `init()` function parameters are:
- `chain_id` (56): Aztec testnet chain ID
- `evm_chain_id` (56): EVM chain ID mapping  
- `receiver_address` ($RECEIVER_ADDRESS): Address that receives message fees
- `token_address` ($TOKEN_CONTRACT_ADDRESS): Token contract for fee payments

The receiver address is the fee collector address, not the Wormhole contract address itself.

### 10. Capture Wormhole Contract address and update addresses.json

#### 10a. Set Wormhole Contract Address

```bash
# Set the wormhole contract address (extract from step 9 deployment logs)
export WORMHOLE_CONTRACT_ADDRESS=<wormhole_contract_address_from_step_9>
```

#### 10b. Update addresses.json

```bash
# Update the addresses.json file with the new wormhole address
jq --arg wormhole "$WORMHOLE_CONTRACT_ADDRESS" '.wormhole = $wormhole' addresses.json > addresses.json.tmp && mv addresses.json.tmp addresses.json
```

**Important**: Wait for the transaction to be mined. You can check the transaction status at [AztecScan](http://aztecscan.xyz/).

## Verification

After deployment, verify that:
- Both contracts are deployed successfully
- Token minting operations completed
- All transactions are confirmed on AztecScan

## Troubleshooting

- **Timeout errors**: These are common on testnet and usually resolve themselves
- **Transaction failures**: Check AztecScan for detailed error messages
- **Network issues**: Ensure you're connected to the correct testnet node