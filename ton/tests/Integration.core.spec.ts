import { Blockchain, BlockchainSnapshot, SandboxContract, TreasuryContract } from '@ton/sandbox';
import { beginCell, Cell, Dictionary, toNano } from '@ton/core';
import { Integrator } from '../wrappers/Integrator';
import '@ton/test-utils';
import { compile } from '@ton/blueprint';
import { Wormhole } from '../wrappers/Wormhole';
import { Crypto, Random, Time, Event } from './TestUtils';
import { createEmptyGuardianSet, decodeCommentPayload, generateVAACell } from '../wrappers/Structs';
import { findTransactionRequired } from '@ton/test-utils';
import { Events, Opcodes, toAnswer } from '../wrappers/Constants';

const NUM_GUARDIANS = 19;

describe('Integrator', () => {
    let integratorCode: Cell;
    let wormholeCode: Cell;

    let blockchain: Blockchain;
    let deployer: SandboxContract<TreasuryContract>;
    let user: SandboxContract<TreasuryContract>;
    let recipient: SandboxContract<TreasuryContract>;
    let integrator: SandboxContract<Integrator>;
    let wormhole: SandboxContract<Wormhole>;
    const keys = Crypto.makeRandomKeyPairs(NUM_GUARDIANS);
    const guardianSetIndex = 0;
    const comment = 'test comment';
    let commentPayloadCell: Cell | undefined = undefined;

    let snapshot1: BlockchainSnapshot;

    beforeAll(async () => {
        integratorCode = await compile('Integrator');
        wormholeCode = await compile('Wormhole');

        blockchain = await Blockchain.create();
        deployer = await blockchain.treasury('deployer');
        user = await blockchain.treasury('user');
        recipient = await blockchain.treasury('recipient');

        const publicKeys = Crypto.mapKeyPairsToXOnlyPublicKeys(keys);
        const guardianSets = createEmptyGuardianSet();
        guardianSets.set(guardianSetIndex, { keys: publicKeys, expirationTime: Time.now(60) });
        wormhole = blockchain.openContract(
            Wormhole.createFromConfig(
                {
                    id: Random.id(16),
                    messageFee: toNano(0.1),
                    sequences: Dictionary.empty(),
                    guardianSets,
                    guardianSetIndex,
                    guardianSetExpiry: Time.now(60),
                    chainId: 0,
                    governanceChainId: 0,
                    governanceContract: Buffer.alloc(32),
                },
                wormholeCode,
            ),
        );

        let deployResult = await wormhole.sendDeploy(deployer.getSender(), toNano('1'));
        expect(deployResult.transactions).toHaveTransaction({
            from: deployer.address,
            to: wormhole.address,
            deploy: true,
            success: true,
        });

        integrator = blockchain.openContract(
            Integrator.createFromConfig(
                {
                    id: Random.id(16),
                    wormholeAddress: wormhole.address,
                },
                integratorCode,
            ),
        );

        deployResult = await integrator.sendDeploy(deployer.getSender(), toNano('0.05'));
        expect(deployResult.transactions).toHaveTransaction({
            from: deployer.address,
            to: integrator.address,
            deploy: true,
            success: true,
        });

        snapshot1 = await blockchain.snapshot();
    });

    beforeEach(async () => {
        blockchain.loadFrom(snapshot1);
    });

    it('should send comment', async () => {
        // the check is done inside beforeEach
        // blockchain and integrator are ready to use
        const result = await integrator.sendComment(user.getSender(), toNano(0.15), {
            queryId: 0xdeadbeef,
            nonce: 0xbadf00d,
            consistencyLevel: 0,
            to: recipient.address,
            comment,
        });
        expect(result.transactions).toHaveTransaction({
            from: user.address,
            to: integrator.address,
            success: true,
            op: Opcodes.OP_SEND_COMMENT,
        });
        expect(result.transactions).toHaveTransaction({
            from: integrator.address,
            to: wormhole.address,
            success: true,
            op: Opcodes.OP_PUBLISH_MESSAGE,
        });
        expect(result.transactions).toHaveTransaction({
            from: wormhole.address,
            to: integrator.address,
            success: true,
            op: toAnswer(Opcodes.OP_PUBLISH_MESSAGE),
        });

        const eventBody = Event.mustFindEvent(
            result.transactions,
            {
                from: integrator.address,
                to: wormhole.address,
                success: true,
                op: Opcodes.OP_PUBLISH_MESSAGE,
            },
            Events.EVENT_MESSAGE_PUBLISHED,
        );
        commentPayloadCell = eventBody.loadRef();
        const commentPayload = decodeCommentPayload(commentPayloadCell);
        expect(commentPayload.to.toString()).toBe(recipient.address.toString());
        expect(commentPayload.comment).toBe(comment);
    });

    it('should relay comment', async () => {
        expect(commentPayloadCell).toBeDefined();

        const result = await integrator.sendRelayComment(user.getSender(), toNano(0.15), {
            queryId: 0xdeadbeef,
            encodedVaa: generateVAACell(19, commentPayloadCell),
        });
        expect(result.transactions).toHaveTransaction({
            from: user.address,
            to: integrator.address,
            success: true,
            op: Opcodes.OP_RELAY_COMMENT,
        });
        expect(result.transactions).toHaveTransaction({
            from: integrator.address,
            to: wormhole.address,
            success: true,
            op: Opcodes.OP_PARSE_AND_VERIFY_VM,
        });
        expect(result.transactions).toHaveTransaction({
            from: wormhole.address,
            to: integrator.address,
            success: true,
            op: toAnswer(Opcodes.OP_PARSE_AND_VERIFY_VM),
        });

        const eventBody = Event.mustFindEvent(
            result.transactions,
            {
                from: wormhole.address,
                to: integrator.address,
                success: true,
                op: toAnswer(Opcodes.OP_PARSE_AND_VERIFY_VM),
            },
            Events.EVENT_VAA_VALIDATED_BY_CORE,
        );
        expect(eventBody.loadBoolean()).toBe(true);

        expect(result.transactions).toHaveTransaction({
            from: integrator.address,
            to: recipient.address,
            success: true,
            op: 0x00000000,
            body: beginCell().storeUint(0x00000000, 32).storeStringRefTail(comment).endCell(),
        });
    });
});

