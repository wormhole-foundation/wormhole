import { deserializeLayout, encoding, Layout } from "@wormhole-foundation/sdk";
import yargs from "yargs";

const wormholeKeyLayout = [
  { name: "tagKey", binary: "uint", size: 1, custom: 0x0a, omit: true },
  { name: "key", binary: "bytes", lengthSize: 1 },
] as const satisfies Layout;

export const command = "parse-guardian-key <key>";
export const desc = "Parse a base64 encoded Wormhole guardian private key";
export const builder = (y: typeof yargs) =>
  y.positional("key", {
    describe: "Base64 encoded guardian private key",
    type: "string",
    demandOption: true,
  });

export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  try {
    const whKey = encoding.b64.decode(argv.key);
    const result = deserializeLayout(wormholeKeyLayout, whKey, { consumeAll: false });
    const { key } = result[0];

    console.log("Successfully deserialized guardian key:");
    console.log(`Hex: ${Buffer.from(key).toString("hex")}`);
  } catch (error: any) {
    console.error("Error deserializing guardian key:", error.message);
    console.error("Make sure the input is a valid base64 encoded guardian key");
    process.exit(1);
  }
};

