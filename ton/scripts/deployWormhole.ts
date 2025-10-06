import { Dictionary, toNano } from '@ton/core';
import { GuardianSetDictionaryValue, Wormhole } from '../wrappers/Wormhole';
import { compile, NetworkProvider } from '@ton/blueprint';
import { Random, Time } from '../tests/TestUtils';
import { Crypto } from '../tests/TestUtils';
import { TON_CHAIN_ID } from '../wrappers/Constants';

export async function run(provider: NetworkProvider) {
    const keys = new Array(19).fill(0).map(() => Crypto.makeRandomKeyPair());
    const publicKeys = keys.map((key) => Crypto.toXOnly(key.keyPair.publicKey as Buffer));

    const guardianSetIndex = 0; // the first guardian set
    const guardianSets = Dictionary.empty(Dictionary.Keys.Uint(8), GuardianSetDictionaryValue);
    guardianSets.set(guardianSetIndex, {
        keys: publicKeys,
        expirationTime: Time.now(Time.hours(24)),
    });

    const wormhole = provider.open(
        Wormhole.createFromConfig(
            {
                id: Random.id(16),
                messageFee: toNano(0.1),
                sequences: Dictionary.empty(),
                guardianSets,
                guardianSetIndex,
                guardianSetExpiry: 0,
                chainId: TON_CHAIN_ID,
                governanceChainId: 0,
                governanceContract: Buffer.alloc(32),
            },
            await compile('Wormhole'),
        ),
    );

    await wormhole.sendDeploy(provider.sender(), toNano(0.1));

    await provider.waitForDeploy(wormhole.address);
}
