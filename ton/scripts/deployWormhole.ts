import { Dictionary, toNano } from '@ton/core';
import { GuardianSetDictionaryValue, Wormhole } from '../wrappers/Wormhole';
import { compile, NetworkProvider } from '@ton/blueprint';
import { makeRandomId, makeRandomKeyPair, toXOnly } from '../tests/TestUtils';

export async function run(provider: NetworkProvider) {
    const keys = new Array(19).fill(0).map(() => makeRandomKeyPair());
    const publicKeys = keys.map((key) => toXOnly(key.keyPair.publicKey as Buffer));

    const guardianSetIndex = 0;
    const guardianSets = Dictionary.empty(Dictionary.Keys.Uint(8), GuardianSetDictionaryValue);
    guardianSets.set(guardianSetIndex, { keys: publicKeys, expirationTime: Math.floor(Date.now() / 1000) + 3600 * 24 });

    const wormhole = provider.open(
        Wormhole.createFromConfig(
            {
                id: makeRandomId(16),
                messageFee: toNano(0.1),
                sequences: Dictionary.empty(),
                guardianSets,
                guardianSetIndex,
                guardianSetExpiry: 0,
                chainId: 0,
                governanceChainId: 0,
                governanceContract: Buffer.alloc(32),
            },
            await compile('Wormhole'),
        ),
    );

    await wormhole.sendDeploy(provider.sender(), toNano(0.1));

    await provider.waitForDeploy(wormhole.address);
}
