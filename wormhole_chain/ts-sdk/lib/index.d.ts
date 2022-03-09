import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { StargateClient, StdFee } from "@cosmjs/stargate";
export declare const TENDERMINT_URL = "http://localhost:26657";
export declare const HOLE_DENOM = "uhole";
export declare function getStargateClient(): Promise<StargateClient>;
export declare function getZeroFee(): StdFee;
export declare function getWallet(mnemonic: string): Promise<DirectSecp256k1HdWallet>;
export declare function getAddress(wallet: DirectSecp256k1HdWallet): Promise<string>;
export declare function executeGovernanceVAA(wallet: DirectSecp256k1HdWallet, hexVaa: string): Promise<any>;
