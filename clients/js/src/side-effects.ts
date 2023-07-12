// <sigh>
// when the native secp256k1 is missing, the eccrypto library decides TO PRINT A MESSAGE TO STDOUT:
// https://github.com/bitchan/eccrypto/blob/a4f4a5f85ef5aa1776dfa1b7801cad808264a19c/index.js#L23
//
// do you use a CLI tool that depends on that library and try to pipe the output
// of the tool into another? tough luck
//
// for lack of a better way to stop this, we patch the console.info function to
// drop that particular message...
// </sigh>
const info = console.info;
console.info = function (x: string) {
  if (x !== "secp256k1 unavailable, reverting to browser version") {
    info(x);
  }
};

const warn = console.warn;
console.warn = function (x: string) {
  if (
    x !==
    "bigint: Failed to load bindings, pure JS will be used (try npm run rebuild?)"
  ) {
    warn(x);
  }
};

// Ensure BigInt can be serialized to json
//
// eslint-disable-next-line @typescript-eslint/no-redeclare
interface BigInt {
  /** Convert to BigInt to string form in JSON.stringify */
  toJSON: () => string;
}
// Without this JSON.stringify() blows up
(BigInt.prototype as any).toJSON = function () {
  return this.toString();
};
