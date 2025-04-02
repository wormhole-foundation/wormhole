# Transfer Verifier -- Integration Tests

## EVM Integration Tests

### Overview

The Transfer Verifier tests involve interacting with the local Ethereum devnet defined by the Tilt setup in this repository.

The basic idea is as follows:
* Interact with the local Ethereum testnet. This should already have important pieces such as the Token Bridge and Core Bridge deployed.
* Use `cast` from the foundry toolset to simulate malicious interactions with the Token Bridge.
* Transfer Verifier detects the malicious messages and emits errors about what went wrong.
* The error messages are logged to a file
* A "monitor" script is used to detect the expected error message, waiting until the file is written to
* If the monitor script sees the expected error message in the error log, it terminates

### Scripts/Components

| Script/Component | Description |
|------------------|-------------|
| `Dockerfile.tx-verifier-evm` | The dockerfile representing the image used for EVM transfer verifier tests. |
| `tx-verifier-evm-runner.sh` | Runs the guardiand binary which contains the transfer verifier tool, which monitors the local Ethereum network for events. |
| `tx-verifier-evm-tests.sh` | Contains the `cast` commands that simulate malicious interactions with the Token Bridge and Core Bridge. It is able to broadcast transactions to the `anvil` instance that powers the Ethereum testnet while being able to impersonate arbitrary senders. <br>This allows performing actions that otherwise should be impossible, like causing a Publish Message event to be emitted from the Core Bridge without a corresponding deposit or transfer into the Token Bridge. <br>The current integration test sends exactly two messages, each one corresponding to a different Token Bridge endpoint (Transfer and Transfer With Payload). |

## Sui Integration Tests

### Overview

The transfer verifier integration tests for Sui also involve interacting with the Sui node deployed as part of the larger Tilt setup. However, owing to the complexities of Sui, injecting the invariants transfer verifier tests for is a lot more intrusive:

* The Sui transfer verifier container is set up such that it can build and deploy a fresh core and token bridge to the Sui node.
* The token bridge's `transfer_tokens` module is modified by the Sui runner script to include two unsafe functions that bypass important security checks.

Similar to EVM, the errors are logged to a file and subsequently read to determine whether or not the malicious actions were detected.

### Scripts/Components

| Script/Component | Description |
|------------------|-------------|
| `Dockerfile.tx-verifier-sui` | The dockerfile representing the image used for Sui transfer verifier tests |
| `sui/tx-verifier-sui-runner.sh` | The runner script that prepares the core and token bridge, launches the transfer verifier for Sui, and initiates and monitors for malicious actions. |
| `sui/sui_config` | The Sui client configuration, copied from the Sui devnet environment in the monorepo. This was done to avoid expanding the scope/context of the dockerfile to additional directories in the monorepo. |
| `sui/transfer_tokens_unsafe.move` | Unsafe variations of the `prepare_transfer` and `transfer_tokens` functions in the Sui `token_bridge` package's `transfer_tokens` module. This file is used within the container to patch the token bridge prior to deployment, and allows injecting the invariants transfer verifier monitors.|