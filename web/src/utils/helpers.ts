import {PublicKey} from "@solana/web3.js";
import {BRIDGE_ADDRESS, WRAPPED_MASTER} from "../config";
import {keccak256} from "ethers/utils";

// derive the ERC20 address of a Solana SPL asset wrapped on ETH.
export function deriveERC20Address(key: PublicKey) {
    let hashData = "0xff" + BRIDGE_ADDRESS.slice(2);
    hashData += keccak256(Buffer.concat([new Buffer([1]), key.toBuffer()])).slice(2) // asset_id
    hashData += keccak256("0x3d602d80600a3d3981f3363d3d373d3d3d363d73" + WRAPPED_MASTER + "5af43d82803e903d91602b57fd5bf3").slice(2) // Bytecode

    return keccak256(hashData).slice(26)
}
