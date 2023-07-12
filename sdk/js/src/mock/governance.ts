import { BN } from "@project-serum/anchor";
import { ChainId, tryNativeToHexString } from "../utils";
import { MockEmitter } from "./wormhole";

const ETHEREUM_KEY_LENGTH = 20;

export class GovernanceEmitter extends MockEmitter {
  constructor(emitterAddress: string, startSequence?: number) {
    super(emitterAddress, 1, startSequence);
  }

  publishGovernanceMessage(
    timestamp: number,
    module: string,
    payload: Buffer,
    action: number,
    chain: number,
    uptickSequence: boolean = true
  ) {
    const serialized = Buffer.alloc(35 + payload.length);

    const moduleBytes = Buffer.alloc(32);
    moduleBytes.write(module, 32 - module.length);
    serialized.write(moduleBytes.toString("hex"), 0, "hex");
    serialized.writeUInt8(action, 32); // action
    serialized.writeUInt16BE(chain, 33);
    serialized.write(payload.toString("hex"), 35, "hex");
    return this.publishMessage(0, serialized, 1, timestamp, uptickSequence);
  }

  publishWormholeSetMessageFee(
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
      "Core",
      payload,
      3,
      chain,
      uptickSequence
    );
  }

  publishWormholeTransferFees(
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
      "Core",
      payload,
      4,
      chain,
      uptickSequence
    );
  }

  publishWormholeGuardianSetUpgrade(
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
      "Core",
      payload,
      2,
      0,
      uptickSequence
    );
  }

  publishWormholeUpgradeContract(
    timestamp: number,
    chain: number,
    newContract: string,
    uptickSequence: boolean = true
  ) {
    const payload = Buffer.alloc(32);
    payload.write(
      tryNativeToHexString(newContract, chain as ChainId),
      0,
      "hex"
    );
    return this.publishGovernanceMessage(
      timestamp,
      "Core",
      payload,
      1,
      chain,
      uptickSequence
    );
  }

  publishTokenBridgeRegisterChain(
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
      "TokenBridge",
      payload,
      1,
      0,
      uptickSequence
    );
  }

  publishTokenBridgeUpgradeContract(
    timestamp: number,
    chain: number,
    newContract: string,
    uptickSequence: boolean = true
  ) {
    const payload = Buffer.alloc(32);
    payload.write(
      tryNativeToHexString(newContract, chain as ChainId),
      0,
      "hex"
    );
    return this.publishGovernanceMessage(
      timestamp,
      "TokenBridge",
      payload,
      2,
      chain,
      uptickSequence
    );
  }

  publishNftBridgeRegisterChain(
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
      "NFTBridge",
      payload,
      1,
      0,
      uptickSequence
    );
  }

  publishNftBridgeUpgradeContract(
    timestamp: number,
    chain: number,
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
      "NFTBridge",
      payload,
      2,
      chain,
      uptickSequence
    );
  }

  publishWormholeRelayerRegisterChain(
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
      "WormholeRelayer",
      payload,
      1,
      0,
      uptickSequence
    );
  }

  publishWormholeRelayerUpgradeContract(
    timestamp: number,
    chain: number,
    newContract: string,
    uptickSequence: boolean = true
  ) {
    const payload = Buffer.alloc(32);
    payload.write(
      tryNativeToHexString(newContract, chain as ChainId),
      0,
      "hex"
    );
    return this.publishGovernanceMessage(
      timestamp,
      "WormholeRelayer",
      payload,
      2,
      chain,
      uptickSequence
    );
  }

  publishWormholeRelayerSetDefaultDeliveryProvider(
    timestamp: number,
    chain: number,
    newRelayProviderAddress: string,
    uptickSequence: boolean = true
  ) {
    const payload = Buffer.alloc(32);
    payload.write(
      tryNativeToHexString(newRelayProviderAddress, chain as ChainId),
      0,
      "hex"
    );
    return this.publishGovernanceMessage(
      timestamp,
      "WormholeRelayer",
      payload,
      3,
      chain,
      uptickSequence
    );
  }
}
