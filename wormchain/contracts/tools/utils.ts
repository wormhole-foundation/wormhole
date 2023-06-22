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
