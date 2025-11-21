# Transfer Verifier

Below are admin controls for the Guardians to configure the Transfer Verifier plugin. For
a background on the feature and its objectives, see [the whitepaper](whitepapers/0014_transfer_verifier.md)

## How To Enable Transfer Verifier
The Transfer Verifier feature is disabled by default. Guardians can enable it by passing the following flag to the `guardiand` command when starting it up:

```bash
# Example 1: Enable Transfer Verifier for chain with ID 2 (Ethereum)
--transferVerifierEnabledChainIDs=2

# Example 2: Enable Transfer Verifier for both Ethereum and Sui
--transferVerifierEnabledChainIDs=2,21
```

This parameter is a comma-separated list of Wormhole Chain IDs for which transfer verification will be enabled.

Only some chains support the Transfer Verifier. If an unsupported chain is specified, the node will not start. It will also display an error message stating that the chain does not have a Transfer Verifier implementation.
