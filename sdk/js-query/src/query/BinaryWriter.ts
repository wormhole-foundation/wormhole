import { Buffer } from "buffer";

// BinaryWriter appends data to the end of a buffer, resizing the buffer as needed
// Numbers are encoded as big endian
export class BinaryWriter {
  private _buffer: Buffer;
  private _offset: number;

  constructor(initialSize: number = 1024) {
    if (initialSize < 0) throw new Error("Initial size must be non-negative");
    this._buffer = Buffer.alloc(initialSize);
    this._offset = 0;
  }

  // Ensure the buffer has the capacity to write `size` bytes, otherwise allocate more memory
  _ensure(size: number) {
    const remaining = this._buffer.length - this._offset;
    if (remaining < size) {
      const oldBuffer = this._buffer;
      const newSize = this._buffer.length * 2 + size;
      this._buffer = Buffer.alloc(newSize);
      oldBuffer.copy(this._buffer);
    }
  }

  writeUint8(value: number) {
    if (value < 0 || value > 255) throw new Error("Invalid value");
    this._ensure(1);
    this._buffer.writeUint8(value, this._offset);
    this._offset += 1;
    return this;
  }

  writeUint16(value: number) {
    if (value < 0 || value > 65535) throw new Error("Invalid value");
    this._ensure(2);
    this._offset = this._buffer.writeUint16BE(value, this._offset);
    return this;
  }

  writeUint32(value: number) {
    if (value < 0 || value > 4294967295) throw new Error("Invalid value");
    this._ensure(4);
    this._offset = this._buffer.writeUint32BE(value, this._offset);
    return this;
  }

  writeUint64(value: bigint) {
    if (value < 0 || value > 18446744073709551615)
      throw new Error("Invalid value");
    this._ensure(8);
    this._offset = this._buffer.writeBigUInt64BE(value, this._offset);
    return this;
  }

  writeUint8Array(value: Uint8Array) {
    this._ensure(value.length);
    this._buffer.set(value, this._offset);
    this._offset += value.length;
    return this;
  }

  data(): Uint8Array {
    const copy = new Uint8Array(this._offset);
    copy.set(this._buffer.subarray(0, this._offset));
    return copy;
  }
}
