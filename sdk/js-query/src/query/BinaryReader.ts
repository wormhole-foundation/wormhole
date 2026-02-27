import { Buffer } from "buffer";
import { uint8ArrayToHex } from "./utils";

// BinaryReader provides the inverse of BinaryWriter
// Numbers are encoded as big endian
export class BinaryReader {
  private _buffer: Buffer;
  private _offset: number;

  constructor(bytes: Uint8Array) {
    // Buffer.from(Uint8Array) copies data respecting byteOffset, unlike
    // Buffer.from(arrayBuffer) which reads from byte 0 of the underlying pool.
    this._buffer = Buffer.from(bytes);
    this._offset = 0;
  }

  readUint8(): number {
    const tmp = this._buffer.readUint8(this._offset);
    this._offset += 1;
    return tmp;
  }

  readUint16(): number {
    const tmp = this._buffer.readUint16BE(this._offset);
    this._offset += 2;
    return tmp;
  }

  readUint32(): number {
    const tmp = this._buffer.readUint32BE(this._offset);
    this._offset += 4;
    return tmp;
  }

  readUint64(): bigint {
    const tmp = this._buffer.readBigUInt64BE(this._offset);
    this._offset += 8;
    return tmp;
  }

  readUint8Array(length: number): Uint8Array {
    const tmp = this._buffer.subarray(this._offset, this._offset + length);
    this._offset += length;
    return new Uint8Array(tmp);
  }

  readHex(length: number): string {
    return uint8ArrayToHex(this.readUint8Array(length));
  }

  readString(length: number): string {
    const tmp = this._buffer
      .subarray(this._offset, this._offset + length)
      .toString();
    this._offset += length;
    return tmp;
  }
}
