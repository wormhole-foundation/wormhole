import { MapLevel } from "../../utils";
import { Network } from "../networks";
import { Chain } from "../chains";

export const nftBridgeContracts = [[
  "Mainnet", [
    ["Solana",    "WnFt12ZrnzZrFZkt2xsNsaNWoQribnuQ5B5FrDbwDhD"],
    ["Ethereum",  "0x6FFd7EdE62328b3Af38FCD61461Bbfc52F5651fE"],
    ["Bsc",       "0x5a58505a96D1dbf8dF91cB21B54419FC36e93fdE"],
    ["Polygon",   "0x90BBd86a6Fe93D3bc3ed6335935447E75fAb7fCf"],
    ["Avalanche", "0xf7B6737Ca9c4e08aE573F75A97B73D7a813f5De5"],
    ["Oasis",     "0x04952D522Ff217f40B5Ef3cbF659EcA7b952a6c1"],
    ["Aurora",    "0x6dcC0484472523ed9Cdc017F711Bcbf909789284"],
    ["Fantom",    "0xA9c7119aBDa80d4a4E0C06C8F4d8cF5893234535"],
    ["Karura",    "0xb91e3638F82A1fACb28690b37e3aAE45d2c33808"],
    ["Acala",     "0xb91e3638F82A1fACb28690b37e3aAE45d2c33808"],
    ["Klaytn",    "0x3c3c561757BAa0b78c5C025CdEAa4ee24C1dFfEf"],
    ["Celo",      "0xA6A377d75ca5c9052c9a77ED1e865Cc25Bd97bf3"],
    ["Aptos",     "0x1bdffae984043833ed7fe223f7af7a3f8902d04129b14f801823e64827da7130"],
    ["Moonbeam",  "0x453cfbe096c0f8d763e8c5f24b441097d577bde2"],
    ["Arbitrum",  "0x3dD14D553cFD986EAC8e3bddF629d82073e188c8"],
    ["Base",      "0xDA3adC6621B2677BEf9aD26598e6939CF0D92f88"],
    ["Optimism",  "0xfE8cD454b4A1CA468B57D79c0cc77Ef5B6f64585"],
  ]], [
  "Testnet", [
    ["Solana",    "2rHhojZ7hpu1zA91nvZmT8TqWWvMcKmmNBCr2mKTtMq4"],
    ["Ethereum",  "0xD8E4C2DbDd2e2bd8F1336EA691dBFF6952B1a6eB"],
    ["Bsc",       "0xcD16E5613EF35599dc82B24Cb45B5A93D779f1EE"],
    ["Polygon",   "0x51a02d0dcb5e52F5b92bdAA38FA013C91c7309A9"],
    ["Avalanche", "0xD601BAf2EEE3C028344471684F6b27E789D9075D"],
    ["Oasis",     "0xC5c25B41AB0b797571620F5204Afa116A44c0ebA"],
    ["Aurora",    "0x8F399607E9BA2405D87F5f3e1B78D950b44b2e24"],
    ["Fantom",    "0x63eD9318628D26BdCB15df58B53BB27231D1B227"],
    ["Karura",    "0x0A693c2D594292B6Eb89Cb50EFe4B0b63Dd2760D"],
    ["Acala",     "0x96f1335e0AcAB3cfd9899B30b2374e25a2148a6E"],
    ["Klaytn",    "0x94c994fC51c13101062958b567e743f1a04432dE"],
    ["Celo",      "0xaCD8190F647a31E56A656748bC30F69259f245Db"],
    ["Moonbeam",  "0x98A0F4B96972b32Fcb3BD03cAeB66A44a6aB9Edb"],
    ["Neon",      "0x66E5BcFD45D2F3f166c567ADa663f9d2ffb292B4"],
    ["Arbitrum",  "0xEe3dB83916Ccdc3593b734F7F2d16D630F39F1D0"],
    ["Optimism",  "0x23908A62110e21C04F3A4e011d24F901F911744A"],
    ["Base",      "0xF681d1cc5F25a3694E348e7975d7564Aa581db59"],
    ["Sepolia",   "0x6a0B52ac198e4870e5F3797d5B403838a5bbFD99"],
    ["Aptos",     "0x1bdffae984043833ed7fe223f7af7a3f8902d04129b14f801823e64827da7130"],
  ]], [
  "Devnet", [
    ["Solana",    "NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA"],
    ["Ethereum",  "0x26b4afb60d6c903165150c6f0aa14f8016be4aec"],
    ["Terra",     "terra1plju286nnfj3z54wgcggd4enwaa9fgf5kgrgzl"],
    ["Bsc",       "0x26b4afb60d6c903165150c6f0aa14f8016be4aec"],
    ["Aptos",     "0x46da3d4c569388af61f951bdd1153f4c875f90c2991f6b2d0a38e2161a40852c"],
  ]],
] as const satisfies MapLevel<Network, MapLevel<Chain, string>>;
