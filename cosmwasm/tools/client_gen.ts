#!/usr/bin/env ts-node
import { TSBuilder } from '@cosmwasm/ts-codegen';

const builder = new TSBuilder({
    contracts: [
        {
            name: "wormhole",
            dir: "../contracts/wormhole/schema"
        },
        {
            name: "cw20-wrapped",
            dir: '../contracts/cw20-wrapped/schema'
        },
        {
            name: 'token-bridge',
            dir: '../contracts/token-bridge/schema'
        },
        {
            name: "accounting",
            dir: "../packages/accounting/schema"
        },
        {
            name: 'wormhole-bindings',
            dir: '../packages/wormhole-bindings/schema'
        },
        {
            name: 'wormchain-accounting',
            dir: '../contracts/wormchain-accounting/schema'
        }
    ],
    outPath: './client/',
    options: {
        bundle: {
            bundleFile: 'index.ts',
            scope: 'contracts'
        },
        types: {
            enabled: true
        },
        client: {
            enabled: true,
            execExtendsQuery: true,
            noImplicitOverride: true
        },
        reactQuery: {
            enabled: false,
        },
        recoil: {
            enabled: false
        },
        messageComposer: {
            enabled: true
        }
    }
})
builder.build().then(() => {
    console.log('✨ all done!');
});
