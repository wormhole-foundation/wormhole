import { BN } from "@project-serum/anchor";
import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { ChainId, tryNativeToHexString } from "../utils";
import { MockEmitter } from "./wormhole";

export class MockTokenBridge extends MockEmitter {
  consistencyLevel: number;

  constructor(
    emitterAddress: string,
    chain: number,
    consistencyLevel: number,
    startSequence?: number
  ) {
    super(emitterAddress, chain, startSequence);
    this.consistencyLevel = consistencyLevel;
  }

  publishTokenBridgeMessage(
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

  publishAttestMeta(
    tokenAddress: string,
    decimals: number,
    symbol: string,
    name: string,
    nonce?: number,
    timestamp?: number,
    uptickSequence: boolean = true
  ) {
    const serialized = Buffer.alloc(100);
    serialized.writeUInt8(2, 0);
    const hexlified = Buffer.from(tokenAddress, "hex");
    if (hexlified.length != 32) {
      throw new Error("tokenAddress must be 32 bytes");
    }
    serialized.write(hexlified.toString("hex"), 1, "hex");
    serialized.writeUInt16BE(this.chain, 33);
    serialized.writeUInt8(decimals, 35);
    // truncate to 32 characters
    symbol = symbol.substring(0, 32);
    serialized.write(symbol, 36);
    // truncate to 32 characters
    name = name.substring(0, 32);
    serialized.write(name, 68);
    return this.publishTokenBridgeMessage(
      serialized,
      nonce,
      timestamp,
      uptickSequence
    );
  }

  serializeTransferOnly(
    withPayload: boolean,
    tokenAddress: string,
    tokenChain: number,
    amount: bigint,
    recipientChain: number,
    recipient: string,
    fee?: bigint,
    fromAddress?: Buffer
  ) {
    const serialized = Buffer.alloc(133);
    serialized.writeUInt8(withPayload ? 3 : 1, 0);
    const amountBytes = new BN(amount.toString()).toBuffer();
    serialized.write(
      amountBytes.toString("hex"),
      33 - amountBytes.length,
      "hex"
    );
    serialized.write(tokenAddress, 33, "hex");
    serialized.writeUInt16BE(tokenChain, 65);
    serialized.write(recipient, 67, "hex");
    serialized.writeUInt16BE(recipientChain, 99);
    if (withPayload) {
      if (fromAddress === undefined) {
        throw new Error("fromAddress === undefined");
      }
      serialized.write(fromAddress.toString("hex"), 101, "hex");
    } else {
      if (fee === undefined) {
        throw new Error("fee === undefined");
      }
      const feeBytes = new BN(fee.toString()).toBuffer();
      serialized.write(feeBytes.toString("hex"), 133 - feeBytes.length, "hex");
    }
    return serialized;
  }

  publishTransferTokens(
    tokenAddress: string,
    tokenChain: number,
    amount: bigint,
    recipientChain: number,
    recipient: string,
    fee: bigint,
    nonce?: number,
    timestamp?: number,
    uptickSequence: boolean = true
  ) {
    return this.publishTokenBridgeMessage(
      this.serializeTransferOnly(
        false, // withPayload
        tokenAddress,
        tokenChain,
        amount,
        recipientChain,
        recipient,
        fee
      ),
      nonce,
      timestamp,
      uptickSequence
    );
  }

  publishTransferTokensWithPayload(
    tokenAddress: string,
    tokenChain: number,
    amount: bigint,
    recipientChain: number,
    recipient: string,
    fromAddress: Buffer,
    payload: Buffer,
    nonce?: number,
    timestamp?: number,
    uptickSequence: boolean = true
  ) {
    return this.publishTokenBridgeMessage(
      Buffer.concat([
        this.serializeTransferOnly(
          true, // withPayload
          tokenAddress,
          tokenChain,
          amount,
          recipientChain,
          recipient,
          undefined, // fee
          fromAddress
        ),
        payload,
      ]),
      nonce,
      timestamp,
      uptickSequence
    );
  }
}

export class MockEthereumTokenBridge extends MockTokenBridge {
  constructor(emitterAddress: string, startSequence?: number) {
    const chain = 2;
    super(
      tryNativeToHexString(emitterAddress, chain as ChainId),
      chain,
      15,
      startSequence
    );
  }

  publishAttestMeta(
    tokenAddress: string,
    decimals: number,
    symbol: string,
    name: string,
    nonce?: number,
    timestamp?: number,
    uptickSequence: boolean = true
  ) {
    return super.publishAttestMeta(
      tryNativeToHexString(tokenAddress, this.chain as ChainId),
      decimals,
      symbol == undefined ? "" : symbol,
      name == undefined ? "" : name,
      nonce,
      timestamp,
      uptickSequence
    );
  }
}

export class MockSolanaTokenBridge extends MockTokenBridge {
  constructor(emitterAddress: PublicKeyInitData) {
    super(new PublicKey(emitterAddress).toBuffer().toString("hex"), 1, 32);
  }

  publishAttestMeta(
    mint: PublicKeyInitData,
    decimals: number,
    symbol?: string,
    name?: string,
    nonce?: number,
    timestamp?: number,
    uptickSequence: boolean = true
  ) {
    return super.publishAttestMeta(
      new PublicKey(mint).toBuffer().toString("hex"),
      decimals,
      symbol == undefined ? "" : symbol,
      name == undefined ? "" : name,
      nonce,
      timestamp,
      uptickSequence
    );
  }
}
