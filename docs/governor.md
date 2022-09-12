# Guardian Chain Governor
Below are admin controls surfaced to the Guardians for the Governor plugin. For
a background on the feature and its objectives, see [the whitepaper](../whitepapers/0007_governor.md).

## Default Behavior / Limits
The Chain Governor feature is disabled by default. Guardians can enable it by passing the following flag to the `guardiand` command when starting it up:

```bash
--chainGovernorEnabled=true
```

To observe the default chain limits, see `node/pkg/governor/mainnet_chains.go`.  Occasionally, these limits will be adjusted to stay in touch with notional drift associated with certain chains going up/down.

### Checking Status

To list the governor status for each chain, Guardians can run the `governor-status` admin command as follows:

```bash
guardiand admin governor-status --socket /path/to/admin.sock
```

When running in the local Tilt-based development environment, the command may be invoked as follows:

```bash
kubectl exec guardian-0 -- /guardiand admin governor-status --socket /tmp/admin.sock
```

The following data will be shown:

1. Chain ID / emitter address
2. Configured limit
3. Value published in the last 24 hours.
4. List of VAAs pending publishing For each VAA, list:
    1. Emitter chain ID and address
    2. Sequence number
    3. Token chain ID and address
    4. Receive time
    5. Value
    6. Release time


For example:

```
chain: solana, dailyLimit: 100, total: 40, numPending: 1
   chain: solana, pending[0], value: 200, vaa: 1/c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f/2, time: 2022-08-09 16:29:39.045960153 +0000 UTC m=+8370.131061963, releaseTime: 2022-08-12 16:29:39.045960153 +0000 UTC m=+8370.131061963
chain: ethereum, dailyLimit: 100000, total: 0, numPending: 0

```

### Releasing VAAs

To manually release a pending VAA (identified by emitted chain ID / address and sequence number), Guardians can run the `governor-release-pending-vaa` admin command as follows:

```bash
guardiand admin governor-release-pending-vaa "emitted_chain_ID/address/sequence_number" --socket /path/to/admin.sock
```

NOTE: VAAs that are published this way will not affect the rolling 24hr limit.

When a VAA is released, it will be placed in a holding area until the next pending VAA check so there may be some delay for it to actually be published.

**Warning:** *Releasing a VAA manually should rarely if ever occur.  If Guardians believe a VAA is not invalid (i.e. resulting from an exploit), they should abstain from releasing VAAs early.  If a super majority of Guardians either (1) abstain or (2) manually release, the VAA will be signed and published once the time delay is met and super majority agrees to sign and publish.*

### Dropping VAAs

To manually remove a pending VAA (identified by emitted chain ID / address and sequence number), Guardians can run the `governor-drop-pending-vaa` admin command as follows:

```bash
guardiand admin governor-drop-pending-vaa "emitted_chain_ID/address/sequence_number" --socket /path/to/admin.sock
```
**Warning:** *Dropping a VAA should only be used in the context of confirmed fraud that directly affects the security of the Wormhole network.  A super minority of Guardians are required to effectively censor a VAA.*

### Resetting Release Timer
To reset the release timer for a specified VAA to `maxEnqueuedTimeInHours` from the current time, Guardians can run the `governor-reset-release-timer` admin command as follows:

```bash
guardiand admin governor-reset-release-timer "emitted_chain_ID/address/sequence_number" --socket /path/to/admin.sock
```

**Warning:** *Resetting a VAA should only be used in the context of needing more time to confirm fraud that directly affects the security of the Wormhole network.  A super minority of Guardians are required to reset the timer for a given VAA.*
