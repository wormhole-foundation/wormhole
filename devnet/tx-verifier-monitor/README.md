# Transfer Verifier -- Integration Tests

## EVM Integration Tests

### Overview

The Transfer Verifier tests involve interacting with the local Ethereum devnet defined by the Tilt set-up in this repository.

The basic idea is as follows:
* Interact with the local Ethereum testnet. This should already have important pieces such as the Token Bridge and Core Bridge deployed.
* Use `cast` from the foundry tool set to simulate malicious interactions with the Token Bridge.
* Transfer Verifier detects the malicious messages and emits errors about what went wrong.
* The error messages are logged to a file
* A "monitor" script is used to detect the expected error message, waiting until the file is written to
* If the monitor script sees the expected error message in the error log, it terminates

## Components

### Scripts

#### transfer-verifier-test.sh

Contains the `cast` commands that simulate malicious interactions with the Token Bridge and Core Bridge. It is able to broadcast
transactions to the `anvil` instance that powers the Ethereum testnet while being able to impersonate arbitrary senders.

This lets us perform actions that otherwise should be impossible, like causing a Publish Message event to be emitted from the Core Bridge
without a corresponding deposit or transfer into the Token Bridge.

The current integration test sends exactly two messages, each one corresponding to a different Token Bridge endpoint
(Transfer and Transfer With Payload).

#### monitor.sh

A bash script that monitors the error log file for a specific error pattern. It runs in an infinite loop so it will
not exit until the error pattern is detected.

The error pattern is defined in `devnet/tx-verifier.yaml` and matches an error string in the Transfer Verifier package.

Once the pattern is detected, a success message is logged to a status file. Currently this is unused but this set-up
could be modified to detect that this script has written the success message to figure out whether the whole test completed successfully.

The integration test is considered successful as soon as two instances of the error pattern are detected, one for
each message type sent by the `transfer-verifier-test.sh`.

### YAML File Description

The YAML file that runs the integration tests runs three containers:
* The Transfer Verifier binary which monitors the state of the local Ethereum network
* The integration test script that generates activity that the Transfer Verifier classifies as malicious
* The monitor script which ensures that the Transfer Verifier successfully
detected the error we expected, and signals to Tilt that the overall test has
succeeded

## Further Work

The tests cover the case where the Transfer Verifier should report when a Message Publication receipt from the 
Token Bridge with a transfer type does not contain any deposits or transfers.

However, the Transfer Verifier can do more than this. It also reports cases where the incoming funds to the Token
Bridge within one receipt are less than the amount encoded in the payload that it sends to the Core Bridge. This
is something like the transfer not being solvent at the resolution of one Ethereum Receipt.

Adding this test would be a good improvement but requires a more complicated test pattern, perhaps combining
multiple transactions into a single call to `cast`.

