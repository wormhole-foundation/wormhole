import {
  Connection,
  PublicKey,
  Commitment,
  PublicKeyInitData,
} from '@solana/web3.js';
import { utils } from '@wormhole-foundation/connect-sdk-solana';
import {
  ChainId,
  toChainId,
  toChainName,
  toNative,
} from '@wormhole-foundation/connect-sdk';

export function deriveEndpointKey(
  tokenBridgeProgramId: PublicKeyInitData,
  emitterChain: number | ChainId,
  emitterAddress: Buffer | Uint8Array | string,
): PublicKey {
  if (emitterChain == toChainId('Solana')) {
    throw new Error(
      'emitterChain == CHAIN_ID_SOLANA cannot exist as foreign token bridge emitter',
    );
  }
  if (typeof emitterAddress == 'string') {
    const parsedAddress = toNative(toChainName(emitterChain), emitterAddress);
    emitterAddress = parsedAddress.toUint8Array();
  }
  return utils.deriveAddress(
    [
      (() => {
        const buf = Buffer.alloc(2);
        buf.writeUInt16BE(emitterChain as number);
        return buf;
      })(),
      emitterAddress,
    ],
    tokenBridgeProgramId,
  );
}

export async function getEndpointRegistration(
  connection: Connection,
  endpointKey: PublicKeyInitData,
  commitment?: Commitment,
): Promise<EndpointRegistration> {
  return connection
    .getAccountInfo(new PublicKey(endpointKey), commitment)
    .then((info) =>
      EndpointRegistration.deserialize(utils.getAccountData(info)),
    );
}

export class EndpointRegistration {
  chain: ChainId;
  contract: Buffer;

  constructor(chain: number, contract: Buffer) {
    this.chain = chain as ChainId;
    this.contract = contract;
  }

  static deserialize(data: Buffer): EndpointRegistration {
    if (data.length != 34) {
      throw new Error('data.length != 34');
    }
    const chain = data.readUInt16LE(0);
    const contract = data.subarray(2, 34);
    return new EndpointRegistration(chain, contract);
  }
}
