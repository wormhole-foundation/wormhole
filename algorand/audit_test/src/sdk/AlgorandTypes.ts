export type Address = string
export type AssetId = number
export type AppId = number
export type UnixTimestamp = number
export type TransactionId = string
export type ContractAmount = bigint

export type Asset = {
    id: AssetId,
    name: string,
    unitName: string,
    decimals: number,
    url: string
}