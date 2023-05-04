export class SuiRpcValidationError extends Error {
  constructor(response: any) {
    super(
      `Sui RPC returned an unexpected response: ${JSON.stringify(response)}`
    );
  }
}
