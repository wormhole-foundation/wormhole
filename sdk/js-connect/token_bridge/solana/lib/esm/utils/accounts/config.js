import { PublicKey, } from '@solana/web3.js';
import { utils } from '@wormhole-foundation/connect-sdk-solana';
export function deriveTokenBridgeConfigKey(tokenBridgeProgramId) {
    return utils.deriveAddress([Buffer.from('config')], tokenBridgeProgramId);
}
export async function getTokenBridgeConfig(connection, tokenBridgeProgramId, commitment) {
    return connection
        .getAccountInfo(deriveTokenBridgeConfigKey(tokenBridgeProgramId), commitment)
        .then((info) => TokenBridgeConfig.deserialize(utils.getAccountData(info)));
}
export class TokenBridgeConfig {
    constructor(wormholeProgramId) {
        this.wormhole = new PublicKey(wormholeProgramId);
    }
    static deserialize(data) {
        if (data.length != 32) {
            throw new Error('data.length != 32');
        }
        const wormholeProgramId = data.subarray(0, 32);
        return new TokenBridgeConfig(wormholeProgramId);
    }
}
