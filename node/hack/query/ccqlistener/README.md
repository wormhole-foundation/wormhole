# CCQ Listener Tools

This directory contains tools for working with Cross Chain Queries (CCQ) in Wormhole.

## Contents

- `ccqlistener.go` - The main CCQ listener implementation
- `Dockerfile` - Used to build the CCQ listener binary
- `deployment/` - Docker Compose setup for easy deployment

## Docker Deployment

The `deployment/` directory provides a complete Docker Compose environment for running the CCQ listener in two modes:

1. **Listen-only mode**: Passively monitors the network for CCQ responses
2. **Query mode**: Actively sends queries and waits for responses

### Using the Deployment Setup

To use the deployment tools:

```bash
# Navigate to the deployment directory
cd node/hack/query/ccqlistener/deployment

# Build the Docker images
./run-ccq.sh build

# Generate a signing key (required for query mode)
./run-ccq.sh genkey

# Start in listen-only mode
./run-ccq.sh listen

# Start in query mode (auto-generates key if needed)
./run-ccq.sh query

# Start in query mode with Ankr API key
./run-ccq.sh query YOUR_API_KEY

# Start in query mode with full RPC URL
./run-ccq.sh query https://rpc.ankr.com/eth/YOUR_API_KEY
```

For more detailed documentation, see the [Deployment README](deployment/README.md).

## Manual Usage

To run the CCQ listener manually, see the comments at the top of `ccqlistener.go` for instructions. 