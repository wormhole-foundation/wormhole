import {
  AccountMeta,
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  SYSVAR_RENT_PUBKEY,
  TransactionInstruction,
} from "@solana/web3.js";
import {
  deriveAddress,
  getAccountData,
  newAccountMeta,
  newReadOnlyAccountMeta,
} from "./account";

export class Creator {
  address: PublicKey;
  verified: boolean;
  share: number;

  constructor(address: PublicKeyInitData, verified: boolean, share: number) {
    this.address = new PublicKey(address);
    this.verified = verified;
    this.share = share;
  }

  static size: number = 34;

  serialize() {
    const serialized = Buffer.alloc(Creator.size);
    serialized.write(this.address.toBuffer().toString("hex"), 0, "hex");
    if (this.verified) {
      serialized.writeUInt8(1, 32);
    }
    serialized.writeUInt8(this.share, 33);
    return serialized;
  }

  static deserialize(data: Buffer): Creator {
    const address = data.subarray(0, 32);
    const verified = data.readUInt8(32) > 0;
    const share = data.readUInt8(33);
    return new Creator(address, verified, share);
  }
}

export class Data {
  name: string;
  symbol: string;
  uri: string;
  sellerFeeBasisPoints: number;
  creators: Creator[] | null;

  constructor(
    name: string,
    symbol: string,
    uri: string,
    sellerFeeBasisPoints: number,
    creators: Creator[] | null
  ) {
    this.name = name;
    this.symbol = symbol;
    this.uri = uri;
    this.sellerFeeBasisPoints = sellerFeeBasisPoints;
    this.creators = creators;
  }

  serialize() {
    const nameLen = this.name.length;
    const symbolLen = this.symbol.length;
    const uriLen = this.uri.length;
    const creators = this.creators;
    const [creatorsLen, creatorsSize] = (() => {
      if (creators === null) {
        return [0, 0];
      }

      const creatorsLen = creators.length;
      return [creatorsLen, 4 + creatorsLen * Creator.size];
    })();
    const serialized = Buffer.alloc(
      15 + nameLen + symbolLen + uriLen + creatorsSize
    );
    serialized.writeUInt32LE(nameLen, 0);
    serialized.write(this.name, 4);
    serialized.writeUInt32LE(symbolLen, 4 + nameLen);
    serialized.write(this.symbol, 8 + nameLen);
    serialized.writeUInt32LE(uriLen, 8 + nameLen + symbolLen);
    serialized.write(this.uri, 12 + nameLen + symbolLen);
    serialized.writeUInt16LE(
      this.sellerFeeBasisPoints,
      12 + nameLen + symbolLen + uriLen
    );
    if (creators === null) {
      serialized.writeUInt8(0, 14 + nameLen + symbolLen + uriLen);
    } else {
      serialized.writeUInt8(1, 14 + nameLen + symbolLen + uriLen);
      serialized.writeUInt32LE(creatorsLen, 15 + nameLen + symbolLen + uriLen);
      for (let i = 0; i < creatorsLen; ++i) {
        const creator = creators.at(i)!;
        const idx = 19 + nameLen + symbolLen + uriLen + i * Creator.size;
        serialized.write(creator.serialize().toString("hex"), idx, "hex");
      }
    }
    return serialized;
  }

  static deserialize(data: Buffer): Data {
    const nameLen = data.readUInt32LE(0);
    const name = data.subarray(4, 4 + nameLen).toString();
    const symbolLen = data.readUInt32LE(4 + nameLen);
    const symbol = data
      .subarray(8 + nameLen, 8 + nameLen + symbolLen)
      .toString();
    const uriLen = data.readUInt32LE(8 + nameLen + symbolLen);
    const uri = data
      .subarray(12 + nameLen + symbolLen, 12 + nameLen + symbolLen + uriLen)
      .toString();
    const sellerFeeBasisPoints = data.readUInt16LE(
      12 + nameLen + symbolLen + uriLen
    );
    const optionCreators = data.readUInt8(14 + nameLen + symbolLen + uriLen);
    const creators = (() => {
      if (optionCreators == 0) {
        return null;
      }

      const creators: Creator[] = [];
      const creatorsLen = data.readUInt32LE(15 + nameLen + symbolLen + uriLen);
      for (let i = 0; i < creatorsLen; ++i) {
        const idx = 19 + nameLen + symbolLen + uriLen + i * Creator.size;
        creators.push(
          Creator.deserialize(data.subarray(idx, idx + Creator.size))
        );
      }
      return creators;
    })();
    return new Data(name, symbol, uri, sellerFeeBasisPoints, creators);
  }
}

export class CreateMetadataAccountArgs extends Data {
  isMutable: boolean;

  constructor(
    name: string,
    symbol: string,
    uri: string,
    sellerFeeBasisPoints: number,
    creators: Creator[] | null,
    isMutable: boolean
  ) {
    super(name, symbol, uri, sellerFeeBasisPoints, creators);
    this.isMutable = isMutable;
  }

  static serialize(
    name: string,
    symbol: string,
    uri: string,
    sellerFeeBasisPoints: number,
    creators: Creator[] | null,
    isMutable: boolean
  ) {
    return new CreateMetadataAccountArgs(
      name,
      symbol,
      uri,
      sellerFeeBasisPoints,
      creators,
      isMutable
    ).serialize();
  }

  static serializeInstructionData(
    name: string,
    symbol: string,
    uri: string,
    sellerFeeBasisPoints: number,
    creators: Creator[] | null,
    isMutable: boolean
  ) {
    return Buffer.concat([
      Buffer.alloc(1, 0),
      CreateMetadataAccountArgs.serialize(
        name,
        symbol,
        uri,
        sellerFeeBasisPoints,
        creators,
        isMutable
      ),
    ]);
  }

  serialize() {
    return Buffer.concat([
      super.serialize(),
      Buffer.alloc(1, this.isMutable ? 1 : 0),
    ]);
  }
}

export class SplTokenMetadataProgram {
  /**
   * @internal
   */
  constructor() {}

  /**
   * Public key that identifies the SPL Token Metadata program
   */
  static programId: PublicKey = new PublicKey(
    "metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s"
  );

  static createMetadataAccounts(
    payer: PublicKey,
    mint: PublicKey,
    mintAuthority: PublicKey,
    name: string,
    symbol: string,
    updateAuthority: PublicKey,
    updateAuthorityIsSigner: boolean = false,
    uri?: string,
    creators?: Creator[] | null,
    sellerFeeBasisPoints?: number,
    isMutable: boolean = false,
    metadataAccount: PublicKey = deriveSplTokenMetadataKey(mint)
  ): TransactionInstruction {
    const keys: AccountMeta[] = [
      newAccountMeta(metadataAccount, false),
      newReadOnlyAccountMeta(mint, false),
      newReadOnlyAccountMeta(mintAuthority, true),
      newReadOnlyAccountMeta(payer, true),
      newReadOnlyAccountMeta(updateAuthority, updateAuthorityIsSigner),
      newReadOnlyAccountMeta(SystemProgram.programId, false),
      newReadOnlyAccountMeta(SYSVAR_RENT_PUBKEY, false),
    ];
    const data = CreateMetadataAccountArgs.serializeInstructionData(
      name,
      symbol,
      uri === undefined ? "" : uri,
      sellerFeeBasisPoints === undefined ? 0 : sellerFeeBasisPoints,
      creators === undefined ? null : creators,
      isMutable
    );
    return {
      programId: SplTokenMetadataProgram.programId,
      keys,
      data,
    };
  }
}

export function deriveSplTokenMetadataKey(mint: PublicKeyInitData): PublicKey {
  return deriveAddress(
    [
      Buffer.from("metadata"),
      SplTokenMetadataProgram.programId.toBuffer(),
      new PublicKey(mint).toBuffer(),
    ],
    SplTokenMetadataProgram.programId
  );
}

export enum Key {
  Uninitialized,
  EditionV1,
  MasterEditionV1,
  ReservationListV1,
  MetadataV1,
  ReservationListV2,
  MasterEditionV2,
  EditionMarker,
}

export class Metadata {
  key: Key;
  updateAuthority: PublicKey;
  mint: PublicKey;
  data: Data;
  primarySaleHappened: boolean;
  isMutable: boolean;

  constructor(
    key: number,
    updateAuthority: PublicKeyInitData,
    mint: PublicKeyInitData,
    data: Data,
    primarySaleHappened: boolean,
    isMutable: boolean
  ) {
    this.key = key as Key;
    this.updateAuthority = new PublicKey(updateAuthority);
    this.mint = new PublicKey(mint);
    this.data = data;
    this.primarySaleHappened = primarySaleHappened;
    this.isMutable = isMutable;
  }

  static deserialize(data: Buffer): Metadata {
    const key = data.readUInt8(0);
    const updateAuthority = data.subarray(1, 33);
    const mint = data.subarray(33, 65);
    const meta = Data.deserialize(data.subarray(65));
    const metaLen = meta.serialize().length;
    const primarySaleHappened = data.readUInt8(65 + metaLen) > 0;
    const isMutable = data.readUInt8(66 + metaLen) > 0;
    return new Metadata(
      key,
      updateAuthority,
      mint,
      meta,
      primarySaleHappened,
      isMutable
    );
  }
}

export async function getMetadata(
  connection: Connection,
  mint: PublicKeyInitData,
  commitment?: Commitment
): Promise<Metadata> {
  return connection
    .getAccountInfo(deriveSplTokenMetadataKey(mint), commitment)
    .then((info) => Metadata.deserialize(getAccountData(info)));
}
