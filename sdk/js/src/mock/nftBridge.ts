import { NodePrivilegedServiceChainGovernorReleasePendingVAADesc } from "@certusone/wormhole-sdk-proto-web/lib/cjs/node/v1/node";
import { BN } from "@project-serum/anchor";
import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { ChainId, tryNativeToHexString } from "../utils";
import { MockEmitter } from "./wormhole";

export class MockNftBridge extends MockEmitter {
  consistencyLevel: number;

  constructor(emitterAddress: string, chain: number, consistencyLevel: number) {
    super(emitterAddress, chain);
    this.consistencyLevel = consistencyLevel;
  }

  publishNftBridgeMessage(
    serialized: Buffer,
    nonce?: number,
    timestamp?: number,
    uptickSequence: boolean = true
  ) {
    return this.publishMessage(
      nonce == undefined ? 0 : nonce,
      serialized,
      this.consistencyLevel,
      timestamp,
      uptickSequence
    );
  }

  publishTransferNft(
    tokenAddress: string,
    tokenChain: number,
    name: string,
    symbol: string,
    tokenId: bigint,
    uri: string,
    recipientChain: number,
    recipient: string,
    nonce?: number,
    timestamp?: number,
    uptickSequence: boolean = true
  ) {
    if (uri.length > 200) {
      throw new Error("uri.length > 200");
    }
    const serialized = Buffer.alloc(166 + uri.length);
    serialized.writeUInt8(1, 0);
    serialized.write(tokenAddress, 1, "hex");
    serialized.writeUInt16BE(tokenChain, 33);
    // truncate to 32 characters
    symbol = symbol.substring(0, 32);
    serialized.write(symbol, 35);
    // truncate to 32 characters
    name = name.substring(0, 32);
    serialized.write(name, 67);
    const tokenIdBytes = new BN(tokenId.toString()).toBuffer();
    serialized.write(
      tokenIdBytes.toString("hex"),
      131 - tokenIdBytes.length,
      "hex"
    );
    serialized.writeUInt8(uri.length, 131);
    serialized.write(uri, 132);
    const uriEnd = 132 + uri.length;
    serialized.write(recipient, uriEnd, "hex");
    serialized.writeUInt16BE(recipientChain, uriEnd + 32);
    return this.publishNftBridgeMessage(
      serialized,
      nonce,
      timestamp,
      uptickSequence
    );
  }
}

export class MockEthereumNftBridge extends MockNftBridge {
  constructor(emitterAddress: string) {
    const chain = 2;
    super(tryNativeToHexString(emitterAddress, chain as ChainId), chain, 15);
  }
}

export class MockSolanaNftBridge extends MockNftBridge {
  constructor(emitterAddress: PublicKeyInitData) {
    super(new PublicKey(emitterAddress).toBuffer().toString("hex"), 1, 32);
  }
}
