import { sign, serialiseVAA, parse, VAA, Payload } from "../../../clients/js/vaa"

export function concatArrays(arrays: Uint8Array[]): Uint8Array {
    const totalLength = arrays.reduce((accum, x) => accum + x.length, 0);
    const result = new Uint8Array(totalLength);

    for (let i = 0, offset = 0; i < arrays.length; i++) {
        result.set(arrays[i], offset);
        offset += arrays[i].length;
    }

    return result;
}
export function encodeUint8(value: number): Uint8Array {
    if (value >= 2 ** 8 || value < 0) {
        throw new Error(`Out of bound value in Uint8: ${value}`);
    }

    return new Uint8Array([value]);
}

export function zeroPadBytes(value: string, length: number) {
    while (value.length < 2 * length) {
        value = "0" + value;
    }
    return value;
}


export function signPayload(
    emitterChain: number,
    emitterAddress: string,
    signers: string[],
    payload: string
): string {
    let parsed = parse(Buffer.from(payload, "hex"))

    let v: VAA<Payload> = {
        version: Number(payload[1]),
        guardianSetIndex: 0,
        signatures: [],
        timestamp: 0,
        nonce: 0,
        emitterChain: emitterChain,
        emitterAddress: emitterAddress,
        sequence: BigInt(0),
        consistencyLevel: 0,
        payload: parsed.payload as Payload
    };
    v.signatures = sign(signers, v);

    return serialiseVAA(v)
}
