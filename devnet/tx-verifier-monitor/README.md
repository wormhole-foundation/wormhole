# Transfer Verifier -- Integration Tests

## EVM Integration Tests

### Overview

The Transfer Verifier tests involve interacting with the local ethereum devnet defined by the Tilt set-up in this repository.

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

#### monitor.sh

A bash script that monitors the error log file for a specific error pattern. It runs in an infinite loop so it will
not exit until the error pattern is detected.

The error pattern is defined in `wormhole/devnet/tx-verifier.yaml` and matches an error string in the Transfer Verifier package.

Once the pattern is detected, a success message is logged to a status file. Currently this is unused but this set-up
could be modified to detect that this script has written the success message to figure out whether the whole test completed successfully.

### Pods

The files detailed below each have a primary role and are responsible for running one of the main pieces of the test functionality:

* The Transfer Verifier binary which monitors the state of the local Ethereum network
* The integration test script that generates activity that the Transfer Verifier classifies as malicious
* The monitor script which ensures that the Transfer Verifier successfully
detected the error we expected, and signals to Tilt that the overall test has
succeeded

#### devnet/tx-verifier.yaml

Runs the Transfer Verifier binary and redirects its STDERR to the error log file. This allows the output of the binary
to be monitored by `monitor.sh`.

#### devnet/tx-verifier-test.yaml

Runs the `transfer-verifier-test.sh` script which simulates malicious Token Bridge activity. Defines the RPC URL used
by that bash script, which corresponds to the `anvil` instance created in the Ethereum devnet.

#### devnet/tx-verifier-monitor.yaml

Defines the expected error string that should be emitted by the Transfer Verifier code assuming that it successfully recognizes
the malicious Token Bridge activity simulated by the `cast` commands in `transfer-verifier-test.sh`.

It also defines a path to the log file that contains this string.
