import { BN } from "@project-serum/anchor";
import { ChainId, tryNativeToHexString } from "../utils";
import { MockEmitter } from "./wormhole";

const ETHEREUM_KEY_LENGTH = 20;

export class GovernanceEmitter extends MockEmitter {
  module: Buffer;

  constructor(emitterAddress: string, module: string) {
    super(emitterAddress, 1);

    this.module = Buffer.alloc(32);
    this.module.write(module, 32 - module.length);
  }

  publishGovernanceMessage(
    timestamp: number,
    payload: Buffer,
    action: number,
    chain: number,
    uptickSequence: boolean = true
  ) {
    const serialized = Buffer.alloc(35 + payload.length);
    serialized.write(this.module.toString("hex"), 0, "hex");
    serialized.writeUInt8(action, 32); // action
    serialized.writeUInt16BE(chain, 33);
    serialized.write(payload.toString("hex"), 35, "hex");
    return this.publishMessage(0, serialized, 1, timestamp, uptickSequence);
  }
}

export class WormholeGovernanceEmitter extends GovernanceEmitter {
  constructor(emitterAddress: string) {
    super(emitterAddress, "Core");
  }

  publishSetMessageFee(
    timestamp: number,
    chain: number,
    amount: bigint,
    uptickSequence: boolean = true
  ) {
    const payload = Buffer.alloc(32);
    const amountBytes = new BN(amount.toString()).toBuffer();
    payload.write(amountBytes.toString("hex"), 32 - amountBytes.length, "hex");
    return this.publishGovernanceMessage(
      timestamp,
      payload,
      3,
      chain,
      uptickSequence
    );
  }

  publishTransferFees(
    timestamp: number,
    chain: number,
    amount: bigint,
    recipient: Buffer,
    uptickSequence: boolean = true
  ) {
    const payload = Buffer.alloc(64);
    const amountBytes = new BN(amount.toString()).toBuffer();
    payload.write(amountBytes.toString("hex"), 32 - amountBytes.length, "hex");
    payload.write(recipient.toString("hex"), 32, "hex");
    return this.publishGovernanceMessage(
      timestamp,
      payload,
      4,
      chain,
      uptickSequence
    );
  }

  publishGuardianSetUpgrade(
    timestamp: number,
    newGuardianSetIndex: number,
    publicKeys: Buffer[],
    uptickSequence: boolean = true
  ) {
    const numKeys = publicKeys.length;
    const payload = Buffer.alloc(5 + ETHEREUM_KEY_LENGTH * numKeys);
    payload.writeUInt32BE(newGuardianSetIndex, 0);
    payload.writeUInt8(numKeys, 4);
    for (let i = 0; i < numKeys; ++i) {
      const publicKey = publicKeys.at(i);
      if (publicKey == undefined) {
        throw Error("publicKey == undefined");
      }
      payload.write(
        publicKey.toString("hex"),
        5 + ETHEREUM_KEY_LENGTH * i,
        "hex"
      );
    }
    return this.publishGovernanceMessage(
      timestamp,
      payload,
      2,
      0,
      uptickSequence
    );
  }
}

export class TokenBridgeGovernanceEmitter extends GovernanceEmitter {
  constructor(emitterAddress: string) {
    super(emitterAddress, "TokenBridge");
  }

  publishRegisterChain(
    timestamp: number,
    chain: number,
    address: string,
    uptickSequence: boolean = true
  ) {
    const payload = Buffer.alloc(34);
    payload.writeUInt16BE(chain, 0);
    payload.write(tryNativeToHexString(address, chain as ChainId), 2, "hex");
    return this.publishGovernanceMessage(
      timestamp,
      payload,
      1,
      0,
      uptickSequence
    );
  }

  publishUpgradeContract(
    timestamp: number,
    newContract: string,
    uptickSequence: boolean = true
  ) {
    const payload = Buffer.alloc(32);
    payload.write(
      tryNativeToHexString(newContract, this.chain as ChainId),
      0,
      "hex"
    );
    return this.publishGovernanceMessage(
      timestamp,
      payload,
      2,
      0,
      uptickSequence
    );
  }
}

export class NftBridgeGovernanceEmitter extends GovernanceEmitter {
  constructor(emitterAddress: string) {
    super(emitterAddress, "NFTBridge");
  }

  publishRegisterChain(
    timestamp: number,
    chain: number,
    address: string,
    uptickSequence: boolean = true
  ) {
    const payload = Buffer.alloc(34);
    payload.writeUInt16BE(chain, 0);
    payload.write(tryNativeToHexString(address, chain as ChainId), 2, "hex");
    return this.publishGovernanceMessage(
      timestamp,
      payload,
      1,
      0,
      uptickSequence
    );
  }

  publishUpgradeContract(
    timestamp: number,
    newContract: string,
    uptickSequence: boolean = true
  ) {
    const payload = Buffer.alloc(32);
    payload.write(
      tryNativeToHexString(newContract, this.chain as ChainId),
      0,
      "hex"
    );
    return this.publishGovernanceMessage(
      timestamp,
      payload,
      2,
      0,
      uptickSequence
    );
  }
}
