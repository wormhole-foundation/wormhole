import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
} from "@solana/web3.js";

export class RegisteredEmitter {
  chain: number;
  contract: number[];

  private constructor(chain: number, contract: number[]) {
    this.chain = chain;
    this.contract = contract;
  }

  private static _legacyAddress(
    programId: PublicKey,
    foreignChain: number,
    foreignEmitter: number[]
  ): PublicKey {
    const encodedChain = Buffer.alloc(2);
    encodedChain.writeUInt16BE(foreignChain, 0);

    return PublicKey.findProgramAddressSync(
      [encodedChain, Buffer.from(foreignEmitter)],
      programId
    )[0];
  }

  private static _address(programId: PublicKey, foreignChain: number): PublicKey {
    const encodedChain = Buffer.alloc(2);
    encodedChain.writeUInt16BE(foreignChain, 0);

    return PublicKey.findProgramAddressSync([encodedChain], programId)[0];
  }

  // NOTE: The foreignEmitter argument is optional because at some point this argument will go away
  // when the registered chain PDA addresses will only be derived by the foreignChain argument.
  static address(programId: PublicKey, foreignChain: number, foreignEmitter?: number[]): PublicKey {
    if (foreignEmitter === undefined) {
      return RegisteredEmitter._address(programId, foreignChain);
    } else {
      return RegisteredEmitter._legacyAddress(programId, foreignChain, foreignEmitter);
    }
  }

  static fromAccountInfo(info: AccountInfo<Buffer>): RegisteredEmitter {
    return RegisteredEmitter.deserialize(info.data);
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<RegisteredEmitter> {
    const accountInfo = await connection.getAccountInfo(address, commitmentOrConfig);
    if (accountInfo == null) {
      throw new Error(`Unable to find RegisteredEmitter account at ${address}`);
    }
    return RegisteredEmitter.fromAccountInfo(accountInfo);
  }

  static async fromPda(
    connection: Connection,
    programId: PublicKey,
    foreignChain: number,
    foreignEmitter?: number[]
  ): Promise<RegisteredEmitter> {
    return RegisteredEmitter.fromAccountAddress(
      connection,
      RegisteredEmitter.address(programId, foreignChain, foreignEmitter)
    );
  }

  static deserialize(data: Buffer): RegisteredEmitter {
    if (data.length != 34) {
      throw new Error("data.length != 34");
    }
    const chain = data.readUInt16LE(0);
    const contract = Array.from(data.subarray(2));
    return new RegisteredEmitter(chain, contract);
  }
}
