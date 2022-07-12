// Borrowed from coral-xyz/anchor
//
// https://github.com/coral-xyz/anchor/blob/master/ts/packages/anchor/src/error.ts

export class IdlError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "IdlError";
  }
}
