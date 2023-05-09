declare module "elliptic" {
  export interface BN {
    length: number;
    negative: number;
    words: Uint8Array;
    toString(format?: string): string;
  }

  export interface Point {
    x: BN;
    y: BN;
  }

  export interface KeyPair {
    getPrivate(): BN;
    getPublic(): Point;
    sign(
      message: Buffer,
      options: any
    ): {
      r: BN;
      s: BN;
      recoveryParam: number;
    };
  }

  export class ec {
    constructor(curveName: string);
    genKeyPair(): KeyPair;
    keyFromPrivate(priv: any, enc?: any): KeyPair;
    keyFromPublic(priv: any, enc: any): KeyPair;
  }
}
