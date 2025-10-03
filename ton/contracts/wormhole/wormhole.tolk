tolk 1.1

import "../libraries/errors";
import "../libraries/tonmsg";
import "events";
import "structs";
import "opcodes";
import "messages";

// this struct defines storage layout of the contract
struct Storage {
    messageFee: uint64
    sequences: map<address, uint64>
    guardianSets: map<uint32, GuardianSet>
    guardianSetIndex: uint32
    guardianSetExpiry: uint32
    chainId: uint16
    governanceChainId: uint16
    governanceContract: uint256
    id: uint16 // unique contract ID
}

// load contract data from the persistent storage
fun Storage.load(): Storage {
    return Storage.fromCell(contract.getData())
}

// save contract data into the persistent storage
fun Storage.save(self) {
    contract.setData(self.toCell())
}

@inline
fun Storage.useSequence(mutate self, sender: address): uint64 {
    val sequence = self.sequences.get(sender);
    if (sequence.isFound) {
        self.sequences.set(sender, sequence.loadValue() + 1);
        return sequence.loadValue();
    } else {
        self.sequences.set(sender, 1);
        return 0;
    }
}

type AllowedMessage = PublishMessage | ParseAndVerifyVM

@inline
fun onPublishMessage(sender: address, msgValue: coins, params: PublishMessage) {
    var storage = lazy Storage.load();
    assert(msgValue > storage.messageFee) throw ERROR_INSUFFICIENT_FEE;

    val sequence = storage.useSequence(sender);

    val event = PublishMessageEvent {
        sender: sender,
        sequence: sequence,
        nonce: params.nonce,
        consistencyLevel: params.consistencyLevel,
        payload: params.payload
    };
    emit(event);
}

@inline
fun quorum(numGuardians: uint8): uint8 {
    return ((numGuardians * 2) / 3) + 1;
}

@inline
fun onParseAndVerifyVM(params: ParseAndVerifyVM) {
    var storage = lazy Storage.load();
    val (vm, vmHash) = VM.parse(params.encodedVM);
    val value = storage.guardianSets.get(vm.guardianSetIndex);
    assert(value.isFound) throw ERROR_INVALID_GUARDIAN_SET;
    val guardianSet = value.loadValue();
    assert (
        vm.guardianSetIndex == storage.guardianSetIndex || 
        guardianSet.expirationTime >= blockchain.now()
    ) throw ERROR_GUARDIAN_SET_EXPIRED;
    assert (vm.signaturesCount >= quorum(guardianSet.count)) throw ERROR_NO_QUORUM;
    val signaturesValid = verifyVmSignatures(vmHash, guardianSet, vm.signatures);
    assert(signaturesValid) throw ERROR_INVALID_VM_SIGNATURES;
}

// the main entrypoint: called when a contract receives 
// internal messages from other contracts
fun onInternalMessage(in: InMessage) {
    // use `lazy` to defer loading fields until they are accessed
    val msg = lazy AllowedMessage.fromSlice(in.body);

    match (msg) {
        PublishMessage => {
            onPublishMessage(in.senderAddress, in.valueCoins, msg);
            sendChange(in.senderAddress, OP_PUBLISH_MESSAGE, msg.queryId, msg.tail);
        }
        ParseAndVerifyVM => {
            onParseAndVerifyVM(msg);
            sendChange(in.senderAddress, OP_PARSE_AND_VERIFY_VM, msg.queryId, true);
        }
        else => {
            // ignore empty messages, "wrong opcode" for others
            assert (in.body.isEmpty()) throw ERROR_UNKNOWN_OPCODE;
        }
    }
}

// a handler for bounced messages (not used here, may be ommited)
fun onBouncedMessage(in: InMessageBounced) {
}

get fun verifyVM(vmCell: cell): bool {
    val storage = lazy Storage.load();
    val (vm, vmHash) = VM.parse(vmCell);
    val signaturesValid = verifyVmSignatures(vmHash, storage.guardianSets.get(vm.guardianSetIndex).loadValue(), vm.signatures);
    return signaturesValid;
}

get fun messageFee(): int {
    val storage = lazy Storage.load();
    return storage.messageFee;
}
