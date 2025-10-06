import { toNano } from '@ton/core';
import { Integrator } from '../wrappers/Integrator';
import { compile, NetworkProvider } from '@ton/blueprint';
import { Random } from '../tests/TestUtils';

export async function run(provider: NetworkProvider) {
    const ui = provider.ui();
    const wormholeAddress = await ui.inputAddress('Wormhole address');
    const integrator = provider.open(
        Integrator.createFromConfig(
            {
                id: Random.id(16),
                wormholeAddress,
            },
            await compile('Integrator'),
        ),
    );

    await integrator.sendDeploy(provider.sender(), toNano(0.1));

    await provider.waitForDeploy(integrator.address);
}
