import { Blockchain, SandboxContract, TreasuryContract } from '@ton/sandbox';
import { Cell, toNano, beginCell, Dictionary } from '@ton/core';
import { Wormhole, GuardianSetDictionaryValue, Events, SignatureDictionaryValue } from '../wrappers/Wormhole';
import '@ton/test-utils';
import { compile } from '@ton/blueprint';
import { makeRandomKeyPair, toXOnly } from './TestUtils';
import { findTransactionRequired } from '@ton/test-utils';
import { randomBytes } from 'crypto';

const NUM_GUARDIANS = 19;
const NUM_SIGNATURES = 13;

describe('Wormhole', () => {
    let code: Cell;

    beforeAll(async () => {
        code = await compile('Wormhole');
    });

    let blockchain: Blockchain;
    let deployer: SandboxContract<TreasuryContract>;
    let publisher: SandboxContract<TreasuryContract>;
    let wormhole: SandboxContract<Wormhole>;

    const keys = new Array(NUM_GUARDIANS).fill(0).map(() => makeRandomKeyPair());

    const generateVM = (signaturesCount: number) => {
        // Create a test VM that follows the contract's parsing order
        const signaturesDict = Dictionary.empty(Dictionary.Keys.Uint(8), SignatureDictionaryValue);
        for (let i = 0; i < signaturesCount; i++) {
            signaturesDict.set(i, { signature: randomBytes(65), guardianIndex: i });
        }
        const vmData = beginCell()
            .storeUint(1, 8) // version
            .storeUint(0, 32) // guardianSetIndex
            .storeUint(signaturesDict.size, 8) // signaturesCount
            .storeDict(signaturesDict)
            .storeUint(Math.floor(Date.now() / 1000), 32) // timestamp
            .storeUint(123, 32) // nonce
            .storeUint(2, 16) // emitterChainId
            .storeUint(0, 256) // emitterAddress
            .storeUint(1, 64) // sequence
            .storeUint(1, 8) // consistencyLevel
            .storeRef(beginCell().storeStringTail('test payload').endCell()) // payload
            .endCell();
        return vmData;
    };

    beforeEach(async () => {
        blockchain = await Blockchain.create();
        deployer = await blockchain.treasury('deployer');
        publisher = await blockchain.treasury('publisher');

        const publicKeys = keys.map((key) => toXOnly(key.keyPair.publicKey as Buffer));

        const guardianSets = Dictionary.empty(Dictionary.Keys.Uint(8), GuardianSetDictionaryValue);
        guardianSets.set(0, { keys: publicKeys, expirationTime: Math.floor(Date.now() / 1000) + 60 });
        wormhole = blockchain.openContract(
            Wormhole.createFromConfig(
                {
                    id: 0,
                    messageFee: toNano(0.1),
                    sequences: Dictionary.empty(),
                    guardianSets,
                    guardianSetIndex: 0,
                    guardianSetExpiry: 0,
                    chainId: 0,
                    governanceChainId: 0,
                    governanceContract: Buffer.alloc(32),
                },
                code,
            ),
        );

        const deployResult = await wormhole.sendDeploy(deployer.getSender(), toNano('1'));

        expect(deployResult.transactions).toHaveTransaction({
            from: deployer.address,
            to: wormhole.address,
            deploy: true,
            success: true,
        });
    });

    it('should succeed getMessageFee', async () => {
        const fee = await wormhole.getMessageFee();
        expect(fee).toBe(toNano('0.1'));
    });

    it('should succeed verifyVM', async () => {
        const vmData = generateVM(NUM_SIGNATURES);
        const result = await wormhole.getVerifyVM(vmData);
        expect(result).toBe(true);
    });

    it('should send publish message with sufficient fee', async () => {
        const messageFee = await wormhole.getMessageFee();

        // Create test payload
        const payload = beginCell().storeUint(0x00000000, 32).storeStringTail('hello, world').endCell();

        const tail = beginCell()
            .storeStringTail('Payload tail')
            .storeRef(beginCell().storeStringTail('this is a reference').endCell())
            .endCell();

        const publishResult = await wormhole.sendPublishMessage(publisher.getSender(), {
            value: messageFee + toNano(0.1),
            queryId: 1,
            nonce: 789,
            consistencyLevel: 1,
            payload,
            tail,
        });

        expect(publishResult.transactions).toHaveTransaction({
            from: publisher.address,
            to: wormhole.address,
            success: true,
        });
        expect(publishResult.transactions).toHaveTransaction({
            from: wormhole.address,
            to: publisher.address,
            success: true,
        });

        const trans = findTransactionRequired(publishResult.transactions, {
            to: wormhole.address,
        });
        const event = trans.outMessages.values().find((msg) => msg.info.type === 'external-out');
        expect(event).toBeDefined();
        const eventBody = event!.body.beginParse();
        expect(eventBody.loadUint(32)).toBe(Events.EVENT_PUBLISH_MESSAGE);
        expect(eventBody.loadAddress().toString()).toBe(publisher.address.toString());
        expect(eventBody.loadUintBig(64)).toBe(0n);
        expect(eventBody.loadUint(32)).toBe(789);
        expect(eventBody.loadUint(8)).toBe(1);
        expect(eventBody.loadRef().hash().toString('hex')).toBe(payload.hash().toString('hex'));
    });

    it('should fail to send publish message with insufficient fee', async () => {
        const messageFee = await wormhole.getMessageFee();

        const payload = beginCell().storeUint(0x00000000, 32).storeStringTail('test payload').endCell();

        const publishResult = await wormhole.sendPublishMessage(publisher.getSender(), {
            value: messageFee - toNano(0.01),
            queryId: 1,
            nonce: 789,
            consistencyLevel: 1,
            payload,
        });

        expect(publishResult.transactions).toHaveTransaction({
            from: publisher.address,
            to: wormhole.address,
            success: false,
            exitCode: 101,
        });
    });

    it('should send parse and verify VM', async () => {
        const verifier = await blockchain.treasury('verifier');
        const vmData = generateVM(NUM_SIGNATURES);
        const verifyResult = await wormhole.sendParseAndVerifyVM(verifier.getSender(), {
            value: toNano(0.1),
            queryId: 1,
            encodedVM: vmData,
        });
        expect(verifyResult.transactions).toHaveTransaction({
            from: verifier.address,
            to: wormhole.address,
            success: true,
        });
    });
});
