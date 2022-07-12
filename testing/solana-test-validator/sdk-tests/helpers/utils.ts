export function now() {
  return Math.floor(Date.now() / 1000);
}

export function ethAddressToBuffer(address: string) {
  return Buffer.concat([
    Buffer.alloc(12),
    Buffer.from(address.substring(2), "hex"),
  ]);
}
