import { AccountsCoder, Idl } from "@project-serum/anchor";
import { accountSize, IdlTypeDef } from "../../anchor";

export class TokenBridgeAccountsCoder<A extends string = string>
  implements AccountsCoder
{
  constructor(private idl: Idl) {}

  public async encode<T = any>(accountName: A, account: T): Promise<Buffer> {
    switch (accountName) {
      default: {
        throw new Error(`Invalid account name: ${accountName}`);
      }
    }
  }

  public decode<T = any>(accountName: A, ix: Buffer): T {
    return this.decodeUnchecked(accountName, ix);
  }

  public decodeUnchecked<T = any>(accountName: A, ix: Buffer): T {
    switch (accountName) {
      default: {
        throw new Error(`Invalid account name: ${accountName}`);
      }
    }
  }

  public memcmp(accountName: A, _appendData?: Buffer): any {
    switch (accountName) {
      default: {
        throw new Error(`Invalid account name: ${accountName}`);
      }
    }
  }

  public size(idlAccount: IdlTypeDef): number {
    return accountSize(this.idl, idlAccount) ?? 0;
  }
}
