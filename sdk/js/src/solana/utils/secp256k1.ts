import { TransactionInstruction, Secp256k1Program } from "@solana/web3.js";

export const SIGNATURE_LENGTH = 65;
export const ETHEREUM_KEY_LENGTH = 20;

/**
 * Create {@link TransactionInstruction} for {@link Secp256k1Program}.
 *
 * @param {Buffer[]} signatures - 65-byte signatures (64 bytes + 1 byte recovery id)
 * @param {Buffer[]} keys - 20-byte ethereum public keys
 * @param {Buffer} message - 32-byte hash
 * @returns Solana instruction for Secp256k1 program
 */
export function createSecp256k1Instruction(
  signatures: Buffer[],
  keys: Buffer[],
  message: Buffer
): TransactionInstruction {
  return {
    keys: [],
    programId: Secp256k1Program.programId,
    data: Secp256k1SignatureOffsets.serialize(signatures, keys, message),
  };
}

/**
 * Secp256k1SignatureOffsets serializer
 *
 * See {@link https://docs.solana.com/developing/runtime-facilities/programs#secp256k1-program} for more info.
 */
export class Secp256k1SignatureOffsets {
  // https://docs.solana.com/developing/runtime-facilities/programs#secp256k1-program
  //
  // struct Secp256k1SignatureOffsets {
  //     secp_signature_key_offset: u16,        // offset to [signature,recovery_id,etherum_address] of 64+1+20 bytes
  //     secp_signature_instruction_index: u8,  // instruction index to find data
  //     secp_pubkey_offset: u16,               // offset to [signature,recovery_id] of 64+1 bytes
  //     secp_signature_instruction_index: u8,  // instruction index to find data
  //     secp_message_data_offset: u16,         // offset to start of message data
  //     secp_message_data_size: u16,           // size of message data
  //     secp_message_instruction_index: u8,    // index of instruction data to get message data
  // }
  //
  // Pseudo code of the operation:
  //
  // process_instruction() {
  //     for i in 0..count {
  //         // i'th index values referenced:
  //         instructions = &transaction.message().instructions
  //         signature = instructions[secp_signature_instruction_index].data[secp_signature_offset..secp_signature_offset + 64]
  //         recovery_id = instructions[secp_signature_instruction_index].data[secp_signature_offset + 64]
  //         ref_eth_pubkey = instructions[secp_pubkey_instruction_index].data[secp_pubkey_offset..secp_pubkey_offset + 32]
  //         message_hash = keccak256(instructions[secp_message_instruction_index].data[secp_message_data_offset..secp_message_data_offset + secp_message_data_size])
  //         pubkey = ecrecover(signature, recovery_id, message_hash)
  //         eth_pubkey = keccak256(pubkey[1..])[12..]
  //         if eth_pubkey != ref_eth_pubkey {
  //             return Error
  //         }
  //     }
  //     return Success
  //   }

  /**
   * Serialize multiple signatures, ethereum public keys and message as Secp256k1 instruction data.
   *
   * @param {Buffer[]} signatures - 65-byte signatures (64 + 1 recovery id)
   * @param {Buffer[]} keys - ethereum public keys
   * @param {Buffer} message - 32-byte hash
   * @returns serialized Secp256k1 instruction data
   */
  static serialize(signatures: Buffer[], keys: Buffer[], message: Buffer) {
    if (signatures.length == 0) {
      throw Error("signatures.length == 0");
    }

    if (signatures.length != keys.length) {
      throw Error("signatures.length != keys.length");
    }

    if (message.length != 32) {
      throw Error("message.length != 32");
    }

    const numSignatures = signatures.length;
    const offsetSpan = 11;
    const dataLoc = 1 + numSignatures * offsetSpan;

    const dataLen = SIGNATURE_LENGTH + ETHEREUM_KEY_LENGTH; // 65 signature size + 20 eth pubkey size
    const messageDataOffset = dataLoc + numSignatures * dataLen;
    const messageDataSize = 32;
    const serialized = Buffer.alloc(messageDataOffset + messageDataSize);

    serialized.writeUInt8(numSignatures, 0);
    serialized.write(message.toString("hex"), messageDataOffset, "hex");

    for (let i = 0; i < numSignatures; ++i) {
      const signature = signatures.at(i);
      if (signature?.length != SIGNATURE_LENGTH) {
        throw Error(`signatures[${i}].length != 65`);
      }

      const key = keys.at(i);
      if (key?.length != ETHEREUM_KEY_LENGTH) {
        throw Error(`keys[${i}].length != 20`);
      }

      const signatureOffset = dataLoc + dataLen * i;
      const ethAddressOffset = signatureOffset + 65;

      serialized.writeUInt16LE(signatureOffset, 1 + i * offsetSpan);
      serialized.writeUInt8(0, 3 + i * offsetSpan);
      serialized.writeUInt16LE(ethAddressOffset, 4 + i * offsetSpan);
      serialized.writeUInt8(0, 6 + i * offsetSpan);
      serialized.writeUInt16LE(messageDataOffset, 7 + i * offsetSpan);
      serialized.writeUInt16LE(messageDataSize, 9 + i * offsetSpan);
      serialized.writeUInt8(0, 11 + i * offsetSpan);

      serialized.write(signature.toString("hex"), signatureOffset, "hex");
      serialized.write(key.toString("hex"), ethAddressOffset, "hex");
    }

    return serialized;
  }
}
