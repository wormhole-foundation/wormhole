# CCQ Listener Deployment

## What's Included

1. **Docker Compose Environment**: 
   - Complete setup for running the CCQ listener
   - Services for listen-only mode, query mode, and key generation

2. **Helper Script**:
   - `run-ccq.sh` for easy running of the different modes
   - Support for building, generating keys, and running in different modes

3. **Working Listen-Only Mode**:
   - Monitors the network for CCQ responses
   - Works without any key setup requirements

4. **Working Query Mode**:
   - Uses a custom key generator that creates compatible Protocol Buffer formatted keys
   - Allows for active queries of the network

## Current Status

- **Listen-Only Mode**: ✅ Working correctly
- **Query Mode**: ✅ Working correctly

## Usage

### Listen-Only Mode

To use the CCQ listener in listen-only mode (passive monitoring):

```bash
./ccq listen
```

This will connect to the Wormhole network and passively monitor for CCQ responses.

### Query Mode

To use the CCQ listener in query mode (active querying):

```bash
./ccq query
```

This will generate a key if needed, connect to the Wormhole network, and send queries to guardians.

> **Important**: For reliable operation, you can provide an Ethereum RPC URL in two ways:
> 
> 1. Using an Ankr API key:
>    ```bash
>    ./ccq query YOUR_API_KEY
>    ```
> 
> 2. Using a full RPC URL:
>    ```bash
>    ./ccq query https://rpc.ankr.com/eth/YOUR_API_KEY
>    ```
>    or any other Ethereum RPC provider:
>    ```bash
>    ./ccq query https://ethereum.publicnode.com
>    ```

## Key Generation

The custom key generator creates keys in the Protocol Buffer format expected by the CCQ listener:

```bash
./ccq genkey
```

This generates a key in the proper format and displays the public key that needs to be registered with the guardians.

## Important Notes

For query mode to work with real guardians, you need to:

1. Register the public key of your generated key with the guardian's `ccqAllowedRequesters` parameter
2. Register your node's peer ID with the guardian's `ccqAllowedPeers` parameter

The peer ID is displayed in the logs when running either mode.

## Conclusion

We've successfully implemented both listen-only and query modes for the CCQ listener. The key generator creates keys in the proper Protocol Buffers format expected by the Wormhole code. This setup provides both passive monitoring and active querying capabilities for the Wormhole Cross Chain Query network. 