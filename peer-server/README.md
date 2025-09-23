## Installation

### Prerequisites

- Node.js 16+ and npm
- Access to an Ethereum RPC endpoint
- Wormhole contract address on the target Ethereum network

### Setup

1. **Clone and navigate to the project directory**:
   ```bash
   cd peer-server
   ```

2. **Install dependencies**:
   ```bash
   npm install
   ```

3. **Configure the server** by editing `config.json`:
   ```json
   {
     "port": 3000,
     "ethereum": {
       "rpcUrl": "https://eth.llamarpc.com",
       "chainId": 1
     },
     "wormholeContractAddress": "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"
   }
   ```

4. **Start the server**:
   ```bash
   npm start
   ```

   Or with auto-reload during development:
   ```bash
   npm run dev
   ```

## Configuration

The server is configured via the `config.json` file in the root directory:

| Field                     | Type     | Description                                   | Required |
|---------------------------|----------|-----------------------------------------------|----------|
| `port`                    | `number` | Port number for the HTTP server (1-65535)     | yes      |
| `ethereum.rpcUrl`         | `string` | Ethereum RPC endpoint URL                     | yes      |
| `ethereum.chainId`        | `number` | Ethereum chain ID (defaults to 1)             | no       |
| `wormholeContractAddress` | `string` | Wormhole Core Bridge contract address         | yes      |

### Configuration Options

You can also specify a custom config file path when starting the server:

```bash
npm start -- --config ./path/to/custom-config.json
```

Or get help with available options:

```bash
npm start -- --help
```

## API Endpoints

### GET /peers

Retrieves all collected peer data from guardians.

**Response:**
```json
{
  "0x1234...abcd": {
    "Hostname": "guardian-1.wormhole.com",
    "TlsX509": "-----BEGIN CERTIFICATE-----\n...",
    "Port": 8999
  },
  "0x5678...efgh": {
    "Hostname": "guardian-2.wormhole.com",
    "TlsX509": "-----BEGIN CERTIFICATE-----\n...",
    "Port": 8999
  }
}
```

### POST /peers

Submits peer data with guardian signature verification.

**Request Body:**
```json
{
  "peer": {
    "Hostname": "guardian-1.wormhole.com",
    "TlsX509": "-----BEGIN CERTIFICATE-----\n...",
    "Port": 8999
  },
  "signature": {
    "signature": "0x1234567890abcdef...",
    "guardianIndex": 0
  }
}
```

**Success Response (201):**
```json
{
  "peer": {
    "Hostname": "guardian-1.wormhole.com",
    "TlsX509": "-----BEGIN CERTIFICATE-----\n...",
    "Port": 8999
  },
  "guardianAddress": "0x1234...abcd"
}
```

**Error Responses:**
- `400 Bad Request`: Missing or invalid peer/signature fields
- `401 Unauthorized`: Invalid guardian signature
- `409 Conflict`: Guardian has already submitted peer data

## Usage

### For Guardians

1. **Prepare your peer data**:
   ```json
   {
     "Hostname": "your-guardian-hostname.com",
     "TlsX509": "your-tls-certificate-content",
     "Port": 8999
   }
   ```

2. **Sign the peer data** with your guardian's private key:
   ```javascript
   const messageHash = ethers.keccak256(
     ethers.solidityPacked(
       ['string', 'string', 'uint256'],
       [hostname, tlsX509, port]
     )
   );
   const signature = await guardianWallet.signMessage(ethers.getBytes(messageHash));
   ```

3. **Submit to the peer server**:
   ```bash
   curl -X POST http://localhost:3000/peers \
     -H "Content-Type: application/json" \
     -d '{
       "peer": {
         "Hostname": "your-hostname",
         "TlsX509": "your-cert",
         "Port": 8999
       },
       "signature": {
         "signature": "0x...",
         "guardianIndex": 0
       }
     }'
   ```

### For Clients

1. **Wait for all guardians to submit** (server will show progress)
2. **Retrieve all peer data**:
   ```bash
   curl http://localhost:3000/peers
   ```

3. **Use the collected peer data** for your Wormhole-related operations

## Development

### Project Structure

```
peer-server/
├── src/
│   ├── index.ts          # Main entry point and configuration
│   ├── server.ts         # Express server and API endpoints
│   ├── wormhole.ts       # Ethereum/Wormhole integration
│   ├── types.ts          # TypeScript type definitions
│   └── display.ts        # Progress display utilities
├── tests/
│   └── server.test.ts    # API endpoint tests
├── config.json           # Server configuration
├── package.json          # Dependencies and scripts
└── README.md             # This file
```

### Available Scripts

- `npm run build` - Compile TypeScript to JavaScript
- `npm start` - Start the production server
- `npm run dev` - Start the server with auto-reload during development
- `npm test` - Run tests in watch mode
- `npm run test:run` - Run tests once

### Testing

The server includes comprehensive tests using Vitest:

```bash
npm test
```

Tests cover:
- API endpoint functionality
- Signature validation
- Error handling
- Configuration loading

## Security

- **Signature Verification**: All peer submissions require valid cryptographic signatures from authorized guardians
- **Guardian Index Validation**: Ensures guardian indices are within the valid range
- **Duplicate Prevention**: Prevents multiple submissions from the same guardian
- **Input Validation**: Validates all required fields and data formats

## Error Handling

The server provides detailed error responses for common issues:
- Invalid signature verification
- Missing or malformed data
- Guardian index out of bounds
- Duplicate submissions
- Network connectivity issues

## Dependencies

- **express**: Web framework for HTTP endpoints
- **ethers**: Ethereum integration and signature verification
- **cors**: Cross-origin resource sharing
- **sqlite3**: Database storage (if needed for future features)

## License

MIT

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## Support

For issues and questions, please refer to the main project documentation or create an issue in the repository.
