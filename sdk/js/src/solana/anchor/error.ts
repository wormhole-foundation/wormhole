// Borrowed from coral-xyz/anchor
//
// https://github.com/otter-sec/anchor/blob/master/ts/packages/anchor/src/error.ts

export class IdlError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "IdlError";
  }
}
