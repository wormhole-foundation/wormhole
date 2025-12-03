import { deserialize, Layout } from "binary-layout";
import { base64 } from "@scure/base";
import { ValidationError } from "./types.js";

export const guardianPrivateKeyArmor = "WORMHOLE GUARDIAN PRIVATE KEY";
export const tlsKeyArmor             = "PRIVATE KEY";
export const tlsCertificateArmor     = "CERTIFICATE";

export type ParsedArmor = {
  headers: string[];
  body: Uint8Array;
};

/**
 * We use this to parse two kinds of armored files: PEM and PGP.
 */
export function parseArmor(input: string, type: string): ValidationError<ParsedArmor> {
  const lines = input.trim().split(/\r?\n/);
  if (!(lines[0] === `-----BEGIN ${type}-----` &&
    lines[lines.length - 1] === `-----END ${type}-----`)) {
    return { success: false, error: `Invalid ${type} armor format` };
  }
  const message = lines.slice(1, lines.length - 1);
  const breakIndex = message.findIndex(line => line.length === 0);

  const headers = breakIndex === -1 ? [] : message.slice(0, breakIndex);
  const lastLine = message[message.length - 1];
  const hasCrc = /^=....$/.test(lastLine);
  const bodyEnd = hasCrc ? message.length - 1 : message.length;
  const body = base64.decode(message.slice(breakIndex + 1, bodyEnd).join("").trim());
  if (hasCrc) {
    const crc = Buffer.from(base64.decode(lastLine.slice(1))).readUintBE(0, 3);
    const expectedCrc = crc24(body);
    if (crc !== expectedCrc) {
      return { success: false, error: `Invalid CRC: ${crc} !== ${expectedCrc}` };
    }
  }
  return { success: true, value: { headers, body } };
}

export function crc24(data: Uint8Array): number {
  let crc = 0xB704CE;
  for (let i = 0; i < data.length; i++) {
    crc ^= data[i] << 16;
    for (let j = 0; j < 8; j++) {
      crc <<= 1;
      if (crc & 0x1000000) {
        crc &= 0xFFFFFF;
        crc ^= 0x864CFB;
      }
    }
  }
  return crc & 0xFFFFFF;
}

export function checkTlsKey(input: string) {
  return parseArmor(input, tlsKeyArmor).success;
}

export function checkTlsCertificate(input: string) {
  return parseArmor(input, tlsCertificateArmor).success;
}

export const wormholeKeyLayout = [
  { name: "tagKey", binary: "uint",  size: 1, custom: 0x0A, omit: true },
  { name: "key",    binary: "bytes", lengthSize: 1 }
] as const satisfies Layout;


export function parseGuardianKey(input: string) {
  const parsed = parseArmor(input, guardianPrivateKeyArmor);
  if (!parsed.success)
    throw new Error(`Guardian private key armor invalid: ${parsed.error}`);
  // There might be other bytes after the key to set metadata flags,
  // thus we set consume all to false.
  const [{key}] = deserialize(wormholeKeyLayout, parsed.value.body, false);
  return key;
}