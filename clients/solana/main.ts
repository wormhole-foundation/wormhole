import yargs from "yargs";

const {hideBin} = require('yargs/helpers')
import * as web3 from '@solana/web3.js';
import {PublicKey, Transaction, TransactionInstruction, AccountMeta, Keypair, Connection} from "@solana/web3.js";

import {setDefaultWasm, importCoreWasm, ixFromRust} from '@certusone/wormhole-sdk'
setDefaultWasm("node")

yargs(hideBin(process.argv))
    .command('post_message [nonce] [message] [consistency]', 'post a message', (yargs) => {
        return yargs
            .positional('nonce', {
                describe: 'nonce of the message',
                type: "number",
                required: true
            })
            .positional('message', {
                describe: 'message to post',
                type: "string",
                required: true
            })
            .positional('consistency', {
                describe: 'confirmation level that this message requires <CONFIRMED|FINALIZED>',
                type: "string",
                required: true
            })
    }, async (argv: any) => {
        const bridge = await importCoreWasm()

        let connection = setupConnection(argv);
        let bridge_id = new PublicKey(argv.bridge);

        // Generate a new random public key
        let from = web3.Keypair.generate();
        let emitter = web3.Keypair.generate();
        let message = web3.Keypair.generate();
        let airdropSignature = await connection.requestAirdrop(
            from.publicKey,
            web3.LAMPORTS_PER_SOL,
        );
        await connection.confirmTransaction(airdropSignature);

        let fee_acc = await bridge.fee_collector_address(bridge_id.toString());
        let bridge_state = await get_bridge_state(connection, bridge_id);
        let transferIx = web3.SystemProgram.transfer({
            fromPubkey: from.publicKey,
            toPubkey: new PublicKey(fee_acc),
            lamports: bridge_state.config.fee,
        });

        if (argv.consistency !== "CONFIRMED" && argv.consistency !== "FINALIZED") {
            throw new Error("invalid consistency level")
        }

        let ix = ixFromRust(bridge.post_message_ix(bridge_id.toString(), from.publicKey.toString(), emitter.publicKey.toString(), message.publicKey.toString(), argv.nonce, Buffer.from(argv.message, "hex"), argv.consistency));
        // Add transfer instruction to transaction
        let transaction = new web3.Transaction().add(transferIx, ix);

        // Sign transaction, broadcast, and confirm
        let signature = await web3.sendAndConfirmTransaction(
            connection,
            transaction,
            [from, emitter, message],
            {
                skipPreflight: true
            }
        );
        console.log('SIGNATURE', signature);
    })
    .command('post_vaa [vaa]', 'post a VAA on Solana', (yargs) => {
        return yargs
            .positional('vaa', {
                describe: 'vaa to post',
                type: "string",
                required: true
            })
    }, async (argv: any) => {
        let connection = setupConnection(argv);
        let bridge_id = new PublicKey(argv.bridge);

        // Generate a new random public key
        let from = web3.Keypair.generate();
        let airdropSignature = await connection.requestAirdrop(
            from.publicKey,
            web3.LAMPORTS_PER_SOL,
        );
        await connection.confirmTransaction(airdropSignature);

        let vaa = Buffer.from(argv.vaa, "hex");
        await post_vaa(connection, bridge_id, from, vaa);
    })
    .command('execute_governance_vaa [vaa]', 'execute a governance VAA on Solana', (yargs) => {
        return yargs
            .positional('vaa', {
                describe: 'vaa to post',
                type: "string",
                required: true
            })
    }, async (argv: any) => {
        const bridge = await importCoreWasm()

        let connection = setupConnection(argv);
        let bridge_id = new PublicKey(argv.bridge);

        // Generate a new random public key
        let from = web3.Keypair.generate();
        let airdropSignature = await connection.requestAirdrop(
            from.publicKey,
            web3.LAMPORTS_PER_SOL,
        );
        await connection.confirmTransaction(airdropSignature);

        let vaa = Buffer.from(argv.vaa, "hex");
        await post_vaa(connection, bridge_id, from, vaa);

        let parsed_vaa = await bridge.parse_vaa(vaa);
        let ix: TransactionInstruction;
        switch (parsed_vaa.payload[32]) {
            case 1:
                console.log("Upgrading contract")
                ix = bridge.upgrade_contract_ix(bridge_id.toString(), from.publicKey.toString(), from.publicKey.toString(), vaa);
                break
            case 2:
                console.log("Updating guardian set")
                ix = bridge.update_guardian_set_ix(bridge_id.toString(), from.publicKey.toString(), vaa);
                break
            case 3:
                console.log("Setting fees")
                ix = bridge.set_fees_ix(bridge_id.toString(), from.publicKey.toString(), vaa);
                break
            case 4:
                console.log("Transferring fees")
                ix = bridge.transfer_fees_ix(bridge_id.toString(), from.publicKey.toString(), vaa);
                break
            default:
                throw new Error("unknown governance action")
        }
        let transaction = new web3.Transaction().add(ixFromRust(ix));

        // Sign transaction, broadcast, and confirm
        let signature = await web3.sendAndConfirmTransaction(
            connection,
            transaction,
            [from],
            {
                skipPreflight: true
            }
        );
        console.log('SIGNATURE', signature);
    })
    .option('rpc', {
        alias: 'u',
        type: 'string',
        description: 'URL of the Solana RPC',
        default: "http://localhost:8899"
    })
    .option('bridge', {
        alias: 'b',
        type: 'string',
        description: 'Bridge address',
        default: "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
    })
    .argv;

async function post_vaa(connection: Connection, bridge_id: PublicKey, payer: Keypair, vaa: Buffer) {
    const bridge = await importCoreWasm()

    let bridge_state = await get_bridge_state(connection, bridge_id);
    let guardian_addr = new PublicKey(bridge.guardian_set_address(bridge_id.toString(), bridge_state.guardian_set_index));
    let acc = await connection.getAccountInfo(guardian_addr);
    if (acc?.data === undefined) {
        return
    }
    let guardian_data = bridge.parse_guardian_set(new Uint8Array(acc?.data));

    let signature_set = Keypair.generate();
    let txs = bridge.verify_signatures_ix(bridge_id.toString(), payer.publicKey.toString(), bridge_state.guardian_set_index, guardian_data, signature_set.publicKey.toString(), vaa)
    // Add transfer instruction to transaction
    for (let tx of txs) {
        let ixs: Array<TransactionInstruction> = tx.map((v: any) => {
            return ixFromRust(v)
        })
        let transaction = new web3.Transaction().add(ixs[0], ixs[1]);

        // Sign transaction, broadcast, and confirm
        await web3.sendAndConfirmTransaction(
            connection,
            transaction,
            [payer, signature_set],
            {
                skipPreflight: true
            }
        );
    }

    let ix = ixFromRust(bridge.post_vaa_ix(bridge_id.toString(), payer.publicKey.toString(), signature_set.publicKey.toString(), vaa));
    let transaction = new web3.Transaction().add(ix);

    // Sign transaction, broadcast, and confirm
    let signature = await web3.sendAndConfirmTransaction(
        connection,
        transaction,
        [payer],
        {
            skipPreflight: true
        }
    );
    console.log('SIGNATURE', signature);
}

async function get_bridge_state(connection: Connection, bridge_id: PublicKey): Promise<BridgeState> {
    const bridge = await importCoreWasm()

    let bridge_state = new PublicKey(bridge.state_address(bridge_id.toString()));
    let acc = await connection.getAccountInfo(bridge_state);
    if (acc?.data === undefined) {
        throw new Error("bridge state not found")
    }
    return bridge.parse_state(new Uint8Array(acc?.data));
}

function setupConnection(argv: yargs.Arguments): web3.Connection {
    return new web3.Connection(
        argv.rpc as string,
        'confirmed',
    );
}

interface BridgeState {
    // The current guardian set index, used to decide which signature sets to accept.
    guardian_set_index: number,

    // Lamports in the collection account
    last_lamports: number,

    // Bridge configuration, which is set once upon initialization.
    config: BridgeConfig,
}

interface BridgeConfig {
    // Period for how long a guardian set is valid after it has been replaced by a new one.  This
    // guarantees that VAAs issued by that set can still be submitted for a certain period.  In
    // this period we still trust the old guardian set.
    guardian_set_expiration_time: number,

    // Amount of lamports that needs to be paid to the protocol to post a message
    fee: number,
}