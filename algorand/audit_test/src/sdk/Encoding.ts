import { decodeAddress, encodeAddress, isValidAddress } from 'algosdk'
import sha512 from "js-sha512"
import { Buffer } from 'buffer'
import { Address } from './AlgorandTypes'
import assert from 'assert'
import { ethers } from 'ethers'

export const TEAL_SIGNATURE_LENGTH = 64
export const SHA256_HASH_LENGTH = 32

export type AlgorandType = bigint | string | boolean | number | Uint8Array

export type IPackedInfoFixed = {
    type: "uint" | "number" | "address" | "double" | "boolean" | "emptyString"
}
export type IPackedInfoVariable = {
    type: "string" | "bytes" | "base64"
    size: number
}
export type IPackedInfoObject = {
    type: "object" | "hash"
    info: IPackedInfo
}

export type IPackedInfoArray = {
    type: "array"
    info: IPackedInfoAny
}

export type IPackedInfoFixedBytes = {
    type: "fixed"
    valueHex: string
}

export type IPackedInfoAny = IPackedInfoFixed | IPackedInfoVariable | IPackedInfoObject | IPackedInfoArray | IPackedInfoFixedBytes
export type IPackedInfo = Record<string, IPackedInfoAny>

export type IStateType = 'uint' | 'bytes'
export type IStateMap = Record<string, IStateType>

export type IStateVar = Uint8Array | bigint
export type IState = Record<string, IStateVar>

// NOTE: !!!! ONLY MODIFY THIS BY APPENDING TO THE END. THE INDEXES EFFECT THE MERKLE LOG HASH VALUES !!!!
export const packedTypeMap = [
    "uint",
    "number",
    "address",
    "double",
    "boolean",
    "string",
    "bytes",
    "base64",
    "object",
    "hash",
    "array",
    "emptyString",
    "fixed",
]

assert(packedTypeMap.length < 128, 'Too many types in packedTypeMap')

export function concatArrays(arrays: Uint8Array[]): Uint8Array {
    const totalLength = arrays.reduce((accum, x) => accum + x.length, 0)
    const result = new Uint8Array(totalLength)

    for (let i = 0, offset = 0; i < arrays.length; i++) {
        result.set(arrays[i], offset)
        offset += arrays[i].length
    }

    return result
}

// Encode the format itself as part of the data for forward compatibility
export function packFormat(format: IPackedInfo): Uint8Array {
    const chunks: Uint8Array[] = []

    // NOTE: Byte-size fields are capped at 128 to allow for future expansion with varints
    // Encode number of fields
    const fieldCount = Object.entries(format).length
    assert(fieldCount < 128, `Too many fields in object: ${fieldCount}`)
    chunks.push(new Uint8Array([fieldCount]))

    for (const [name, type] of Object.entries(format)) {
        // Encode name and type index
        assert(name.length < 128, `Name of property ${name} too long`)
        chunks.push(new Uint8Array([name.length]))
        chunks.push(encodeString(name))

        const typeIndex = packedTypeMap.indexOf(type.type)
        assert(typeIndex >= 0, 'Type index not found in packedTypeMap')

        chunks.push(new Uint8Array([typeIndex]))

        // For complex types, encode additional data
        switch (type.type) {
            case "string":
            case "bytes":
            case "base64":
                assert(type.size < 128, `Sized data was too large: ${type.size}`)
                chunks.push(new Uint8Array([type.size]))
                break

            case "hash":
            case "object":
            case "array": {
                const format = packFormat(type.type === 'array' ? { value: type.info } : type.info)
                chunks.push(encodeUint64(format.length))
                chunks.push(format)
                break
            }
        }
    }

    return concatArrays(chunks)
}

export function unpackFormat(data: Uint8Array): IPackedInfo {
    let index = 0
    // Decode field count
    const fieldCount = data[index]
    index++

    const format: IPackedInfo = {}
    for (let i = 0; i < fieldCount; i++) {
        // Decode name
        const nameLen = data[index]
        index++

        const name = decodeString(data.slice(index, index + nameLen))
        index += nameLen

        // Decode type
        const type = packedTypeMap[data[index]]
        index++

        switch (type) {
            case "uint":
            case "number":
            case "address":
            case "double":
            case "boolean":
            case "emptyString":
                format[name] = { type }
                break

            case "string":
            case "bytes":
            case "base64": {
                const size = data[index]
                index++

                format[name] = { type, size }
                break
            }

            case "object":
            case "hash":
            case "array": {
                const length = Number(decodeUint64(data.slice(index, index + 8)))
                index += 8

                const info = unpackFormat(data.slice(index, index + length))
                index += length

                if (type === "array") {
                    format[name] = { type, info: info.value }
                } else {
                    format[name] = { type, info }
                }

                break
            }
        }
    }

    return format
}

export function packData(value: Record<string, any>, format: IPackedInfo, includeType = false): Uint8Array {
    const chunks: Uint8Array[] = []

    if (includeType) {
        const packedFormat = packFormat(format)
        chunks.push(encodeUint64(packedFormat.length))
        chunks.push(packedFormat)
    }

    // Encode the data fields
    for (const [name, type] of Object.entries(format)) {
        const v = value[name]
        if (v === undefined && type.type !== 'fixed') {
            throw new Error(`Key "${name}" missing from value:\n${value.keys}`)
        }

        switch (type.type) {
            case 'object':
                if (v instanceof Object) {
                    chunks.push(packData(v, type.info, false))
                    break
                } else {
                    throw new Error(`${name}: Expected object, got ${v}`)
                }
            case 'hash':
                if (v instanceof Object) {
                    // NOTE: Hashes always refer to the typed version of the data to enable forward compatibility
                    chunks.push(sha256Hash(packData(v, type.info, true)))
                    break
                } else {
                    throw new Error(`${name}: Expected object for hashing, got ${v}`)
                }
            case 'array':
                if (v instanceof Array) {
                    assert(v.length < 128, `Array too large to be encoded: ${v}`)
                    chunks.push(new Uint8Array([v.length]))
                    v.forEach((value) => {
                        chunks.push(packData({ value }, { value: type.info }, false))
                    })
                    break
                } else {
                    throw new Error(`${name}: Expected array, got ${v}`)
                }

            case 'address':
                if (v instanceof Uint8Array) {
                    if (v.length === 20) {
                        const newValue = new Uint8Array(32)
                        newValue.set(new TextEncoder().encode("EthereumAddr"))
                        newValue.set(v, 12)
                        chunks.push(newValue)
                    } else if (v.length === 32) {
                        chunks.push(v)
                    } else {
                        throw new Error(`Invalid address byte array length ${v.length}, expected 20 or 32`)
                    }
                } else if (typeof v === 'string') {
                    if (ethers.utils.isAddress(v)) {
                        const newValue = new Uint8Array(32)
                        newValue.set(new TextEncoder().encode("EthereumAddr"))
                        newValue.set(Buffer.from(v.slice(2), 'hex'), 12)
                        chunks.push(newValue)
                    } else if (isValidAddress(v)) {
                        chunks.push(decodeAddress(v).publicKey)
                    } else {
                        throw new Error(`Invalid address string ${v}`)
                    }
                } else {
                    throw new Error(`${name}: Expected address, got ${v}`)
                }

                break

            case 'bytes':
                if (v instanceof Uint8Array) {
                    if (v.length === type.size) {
                        chunks.push(v)
                        break
                    } else {
                        throw new Error(`${name}: Bytes length is wrong, expected ${type.size}, got ${v.length}`)
                    }
                } else {
                    throw new Error(`${name}: Expected bytes[${type.size}], got ${v}`)
                }
            case 'base64':
                if (typeof v === 'string') {
                    try {
                        const bytes = decodeBase64(v)
                        if (bytes.length === type.size) {
                            chunks.push(bytes)
                            break
                        } else {
                            throw new Error(`${name}: Base64 length is wrong, expected ${type.size}, got ${bytes.length}`)
                        }
                    } catch {
                        throw new Error(`${name}: Base64 encoding is wrong, got ${v}`)
                    }
                } else {
                    throw new Error(`${name}: Expected Base64 string, got ${v}`)
                }
            case 'double':
                if (typeof v === 'number') {
                    const bytes = new ArrayBuffer(8)
                    Buffer.from(bytes).writeDoubleLE(v, 0)
                    chunks.push(new Uint8Array(bytes))
                    break
                } else {
                    throw new Error(`${name}: Expected double, got ${v}`)
                }
            case 'boolean':
                if (typeof v === 'boolean') {
                    chunks.push(new Uint8Array([v ? 1 : 0]))
                    break
                } else {
                    throw new Error(`${name}: Expected boolean, got ${v}`)
                }
            case 'number':
            case 'uint':
                if (typeof v === 'bigint' || typeof v === 'number') {
                    chunks.push(encodeUint64(v))
                    break
                } else {
                    throw new Error(`${name}: Expected uint or number, got ${v}`)
                }
            case 'string':
                if (typeof v === 'string') {
                    const str = encodeString(v)
                    if (str.length === type.size) {
                        chunks.push(str)
                        break
                    } else {
                        throw new Error(`${name}: Expected string length ${type.size}, got string length ${str.length}`)
                    }
                } else {
                    throw new Error(`${name}: Expected string length ${type.size}, got ${v}`)
                }
            case 'emptyString':
                if (typeof v === 'string') {
                    break
                } else {
                    throw new Error(`${name}: Expected string, got ${v}`)
                }
            case 'fixed':
                chunks.push(decodeBase16(type.valueHex))
                break
        }
    }

    return concatArrays(chunks)
}

export function unpackData(data: Uint8Array, formatOpt?: IPackedInfo): Record<string, any> {
    let format: IPackedInfo
    let index = 0

    // Decode format
    if (formatOpt) {
        format = formatOpt
    } else {
        const length = Number(decodeUint64(data.slice(index, index + 8)))
        index += 8

        format = unpackFormat(data.slice(index, index + length))
        index += length
    }

    // Decode data
    // NOTE: This needs to be an inner function to maintain the index across calls
    const unpackInner = (data: Uint8Array, format: IPackedInfo) => {
        const object: Record<string, any> = {}
        for (const [name, type] of Object.entries(format)) {
            if (index >= data.length) {
                throw new Error(`Unpack data length was not enough for the format provided. Data: ${data}, format: ${JSON.stringify(format)}`)
            }

            let value: any
            switch (type.type) {
                case 'object':
                    value = unpackInner(data, type.info)
                    break
                case 'hash':
                    value = new Uint8Array(data.slice(index, index + SHA256_HASH_LENGTH))
                    index += SHA256_HASH_LENGTH
                    break
                case 'array': {
                    const count = data[index++]
                    value = []
                    for (let i = 0; i < count; i++) {
                        value.push(unpackInner(data, { value: type.info }).value)
                    }
                    break
                }
                case 'address':
                    value = encodeAddress(data.slice(index, index + 32))
                    index += 32
                    break
                case 'bytes':
                    value = new Uint8Array(data.slice(index, index + type.size))
                    index += type.size
                    break
                case 'base64':
                    value = encodeBase64(data.slice(index, index + type.size))
                    index += type.size
                    break
                case 'double':
                    value = Buffer.from(data.slice(index, index + 8)).readDoubleLE(0)
                    index += 8
                    break
                case 'boolean':
                    value = data.slice(index, index + 1)[0] === 1
                    index += 1
                    break
                case 'number':
                    value = Number(decodeUint64(data.slice(index, index + 8)))
                    index += 8
                    break
                case 'uint':
                    value = decodeUint64(data.slice(index, index + 8))
                    index += 8
                    break
                case 'string':
                    value = decodeString(data.slice(index, index + type.size))
                    index += type.size
                    break
                case 'emptyString':
                    value = "" 
                    break
                case 'fixed':
                    value = decodeBase16(type.valueHex)
                    break
                default:
                    throw new Error(`Unknown decode type: ${type}`)
            }

            object[name] = value
        }

        return object
    }

    const result = unpackInner(data, format)

    if (index !== data.length) {
        throw new Error(`Data consumed(${index} bytes) did not match expected (${data.length} bytes) for format\nFormat: ${JSON.stringify(format)}\nValue: ${Buffer.from(data).toString('hex')}`)
    }

    return result
}

export function encodeArgArray(params: AlgorandType[]): Uint8Array[] {
    return params.map(param => {
        if (param instanceof Uint8Array)
            return new Uint8Array(param)
        if (typeof param === "string")
            return encodeString(param)
        if (typeof param === "boolean")
            param = BigInt(param ? 1 : 0)
        if (typeof param === "number")
            param = BigInt(param)
        return encodeUint64(param)
    })
}

export function encodeString(value: string | Uint8Array): Uint8Array {
    return new Uint8Array(Buffer.from(value))
}

export function decodeString(value: Uint8Array): string {
    return Buffer.from(value).toString('utf-8')
}

export function decodeState(state: Record<string, Record<string, string>>[], stateMap: IStateMap, errorOnMissing = true): IState {
    const result: IState = {}
    for (const [name, type] of Object.entries(stateMap)) {
        const stateName = encodeBase64(encodeString(name))
        const key = state.find((v: any) => v['key'] === stateName)
        if (errorOnMissing && key === undefined) {
            throw new Error(`Expected key ${name} was not found in state`)
        }

        const value = key ? key['value'][type] : undefined
        if (errorOnMissing && value === undefined) {
            throw new Error(`Expected value for key ${name} was not found in state`)
        }

        const typedValue = type === 'bytes' ? decodeBase64(value ?? '') : BigInt(value ?? '')
        result[name] = typedValue
    }
    return result
}

export function encodeUint64(value: number | bigint): Uint8Array {
    const bytes: Buffer = Buffer.alloc(8)
    for (let index = 0; index < 8; index++)
        bytes[7 - index] = Number((BigInt(value) >> BigInt(index * 8)) & BigInt(0xFF))
    return new Uint8Array(bytes)
}

export function decodeUint64(value: Uint8Array): bigint {
    assert(value.length >= 8, `Expected at least 8 bytes to decode a uint64, but got ${value.length} bytes\nValue: ${Buffer.from(value).toString('hex')}`)

    let num = BigInt(0)
    for (let index = 0; index < 8; index++) {
        num = (num << BigInt(8)) | BigInt(value[index])
    }

    return num
}

export function encodeUint32(value: number): Uint8Array {
    if (value >= 2 ** 32 || value < 0) {
        throw new Error(`Out of bound value in Uint16: ${value}`)
    }

    const bytes: Buffer = Buffer.alloc(4)
    for (let index = 0; index < 4; index++)
        bytes[3 - index] = Number((BigInt(value) >> BigInt(index * 8)) & BigInt(0xFF))
    return new Uint8Array(bytes)
}

export function decodeUint32(value: Uint8Array): number {
    let num = BigInt(0)
    for (let index = 0; index < 4; index++)
        num = (num << BigInt(8)) | BigInt(value[index])
    return Number(num)
}

export function encodeUint16(value: number): Uint8Array {
    if (value >= 2 ** 16 || value < 0) {
        throw new Error(`Out of bound value in Uint16: ${value}`)
    }

    return new Uint8Array([value >> 8, value & 0xFF])
}

export function decodeUint16(value: Uint8Array): number {
    if (value.length !== 2) {
        throw new Error(`Invalid value length, expected 2, got ${value.length}`)
    }

    return value[0] * 256 + value[1]
}

export function encodeUint8(value: number): Uint8Array {
    if (value >= 2 ** 8 || value < 0) {
        throw new Error(`Out of bound value in Uint8: ${value}`)
    }

    return new Uint8Array([value])
}

export function decodeBase16(value: string): Uint8Array {
    return Buffer.from(value, 'hex')
}

export function encodeBase64(value: Uint8Array): string {
    return Buffer.from(value).toString('base64')
}

export function decodeBase64(value: string): Uint8Array {
    return Buffer.from(value, 'base64')
}

export function sha256Hash(arr: sha512.Message): Uint8Array {
    return new Uint8Array(sha512.sha512_256.arrayBuffer(arr))
}

export function encodeApplicationAddress(id: number): Address {
    const APP_ID_PREFIX = Buffer.from('appID');
    const toBeSigned = concatArrays([APP_ID_PREFIX, encodeUint64(BigInt(id))]);
    return encodeAddress(sha256Hash(toBeSigned));
}

export function compareArrays(a: Uint8Array[], b: Uint8Array[]) {
    return (a===undefined || b===undefined) ? a===b : (a.length === b.length && a.reduce((equal, item, index) => equal && item===b[index], true))
}

function getDelta(response: any, key: string): any | undefined {
    const delta = response['global-state-delta'].find((v: any) => v.key === key)
    if (delta === undefined)
        return undefined
    return delta['value']
}

export function getDeltaUint(response: any, key: string): bigint | undefined {
    const delta = getDelta(response, key)
    if (delta === undefined)
        return undefined
    return BigInt(delta['uint'])
}

export function getDeltaBytes(response: any, key: string): Uint8Array | undefined {
    const delta = getDelta(response, key)
    if (delta === undefined)
        return undefined
    return decodeBase64(delta['bytes'])
}

export { encodeAddress } from 'algosdk'
