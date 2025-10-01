import { deserializeLayout, encoding, Layout } from "@wormhole-foundation/sdk";

const guardianPrivateKeyArmor = "WORMHOLE GUARDIAN PRIVATE KEY";
const tlsKeyArmor             = "PRIVATE KEY";
const tlsCertificateArmor     = "CERTIFICATE";

type ParseResult = { valid: false } | { valid: true; headers: string[]; body: string };

/**
 * We use this to parse two kinds of armored files: PEM and PGP.
 */
function parseArmor(input: string, type: string): ParseResult {
  const lines = input.trim().split(/\r?\n/);
  const message = lines.slice(1, lines.length - 2);
  let headers: string[] | undefined, body;
  for (let i = 0; i < message.length; ++i) {
    const line = message[i];
    if (line.length > 0) continue;

    headers = message.slice(0, i + 1);
    body = message.slice(i + 1).join("").trim();
    break;
  }
  if (headers === undefined) headers = [];
  if (body === undefined)    body = message.join("").trim();

  return {
    valid: lines[0] === `-----BEGIN ${type}-----` &&
      lines[lines.length - 1] === `-----END ${type}-----`,
    headers,
    body,
  };
}

function checkTlsKey(input: string) {
  return parseArmor(input, tlsKeyArmor).valid;
}

function checkTlsCertificate(input: string) {
  return parseArmor(input, tlsCertificateArmor).valid;
}


const wormholeKeyLayout = [
  { name: "tagKey", binary: "uint",  size: 1, custom: 0x0A, omit: true},
  { name: "key",    binary: "bytes", lengthSize: 1 }
] as const satisfies Layout;


function parseGuardianKey(input: string) {
  const parsed = parseArmor(input, guardianPrivateKeyArmor);

  if (!parsed.valid) throw new Error(`Guardian private key armor invalid!`);

  const whKey = encoding.b64.decode(parsed.body);
  // There might be other bytes after the key to set metadata flags,
  // thus we set consume all to false.
  const [{key}] = deserializeLayout(wormholeKeyLayout, whKey, false);
  return key;
}
