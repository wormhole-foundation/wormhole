import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
} from "@solana/web3.js";

export class WrappedAsset {
  tokenChain: number;
  tokenAddress: number[];
  nativeDecimals: number;

  private constructor(tokenChain: number, tokenAddress: number[], nativeDecimals: number) {
    this.tokenChain = tokenChain;
    this.tokenAddress = tokenAddress;
    this.nativeDecimals = nativeDecimals;
  }

  static address(programId: PublicKey, mint: PublicKey): PublicKey {
    return PublicKey.findProgramAddressSync([Buffer.from("meta"), mint.toBuffer()], programId)[0];
  }

  static fromAccountInfo(info: AccountInfo<Buffer>): WrappedAsset {
    return WrappedAsset.deserialize(info.data);
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<WrappedAsset> {
    const accountInfo = await connection.getAccountInfo(address, commitmentOrConfig);
    if (accountInfo == null) {
      throw new Error(`Unable to find BridgeProgramData account at ${address}`);
    }
    return WrappedAsset.fromAccountInfo(accountInfo);
  }

  static async fromPda(
    connection: Connection,
    programId: PublicKey,
    mint: PublicKey
  ): Promise<WrappedAsset> {
    return WrappedAsset.fromAccountAddress(connection, WrappedAsset.address(programId, mint));
  }

  static deserialize(data: Buffer): WrappedAsset {
    if (data.length != 35) {
      throw new Error("data.length != 35");
    }
    const tokenChain = data.readUInt16LE(0);
    const tokenAddress = Array.from(data.subarray(2, 34));
    const nativeDecimals = data.readUInt8(34);
    return new WrappedAsset(tokenChain, tokenAddress, nativeDecimals);
  }
}
