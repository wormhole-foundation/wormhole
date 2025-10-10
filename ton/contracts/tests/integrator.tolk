tolk 1.1

import "../libraries/errors";
import "../libraries/tonmsg";
import "../wormhole/messages";
import "../wormhole/structs";

/*
Opcodes generated with the most significant bit set to 0 to distinguish 
it from the response opcode, which has the most significant bit set to 1.
Remark: we cannot use constants as struct tags (only literals in tolk 1.1)
so these constants are unused.
*/
const OP_SEND_COMMENT = stringCrc32("op::integrator::send_comment");
const OP_RELAY_COMMENT = stringCrc32("op::integrator::Relay_comment");

const ERROR_INVALID_TAIL = 1000;
const CHAIN_ID_TON = 62;


struct Storage {
    wormholeAddress: address
    nonce: uint32
    id: uint16
}

// load contract data from the persistent storage
fun Storage.load(): Storage {
    return Storage.fromCell(contract.getData())
}

// save contract data into the persistent storage
fun Storage.save(self) {
    contract.setData(self.toCell())
}

// Call on source chain to send a comment
struct (0x222A627E) SendComment {
    queryId: uint64
    consistencyLevel: uint8
    chainId: uint16
    to: uint256
    comment: cell
}

// Call on destination chain to relay a comment to the recipient
struct (0x327587B5) RelayComment {
    queryId: uint64
    encodedVaa: cell
}

struct CommentVaa {
    chainId: uint16
    to: uint256
    comment: cell
}

struct (0x00000001) VAAValidatedByCoreEvent {
    valid: bool
}

type Operations = SendComment | PublishMessageAnswer | RelayComment | ParseAndVerifyVMAnswer

fun onInternalMessage(in: InMessage) {
    val msg = lazy Operations.fromSlice(in.body);
    match (msg) {
        SendComment => {
            var storage = lazy Storage.load();
            val payload = PublishMessage {
                queryId: msg.queryId,
                nonce: storage.nonce,
                consistencyLevel: msg.consistencyLevel,
                payload: CommentVaa {
                    chainId: msg.chainId,
                    to: msg.to,
                    comment: msg.comment
                }.toCell(),
                tail: beginCell().storeUint(0x11223344, 32).endCell()
            };

            storage.nonce += 1;
            storage.save();
            
            call(storage.wormholeAddress, 0, SEND_MODE_CARRY_ALL_REMAINING_MESSAGE_VALUE, payload);
        }
        PublishMessageAnswer => {
            val tailHash = beginCell().storeUint(0x11223344, 32).endCell().hash();
            assert(msg.tail.hash() == tailHash) throw ERROR_INVALID_TAIL;
        }
        RelayComment => {
            var storage = lazy Storage.load();
            val payload = ParseAndVerifyVM {
                queryId: msg.queryId,
                encodedVM: msg.encodedVaa,
                tail: beginCell().storeUint(0x11223344, 32).endCell()
            };
            call(storage.wormholeAddress, 0, SEND_MODE_CARRY_ALL_REMAINING_MESSAGE_VALUE, payload);
        }
        ParseAndVerifyVMAnswer => {
            emit(VAAValidatedByCoreEvent{ valid: msg.valid });
            val (vaa, vaaHash) = VM.parse(msg.encodedVaa);
            val comment = CommentVaa.fromCell(vaa.payload);
            assert(comment.chainId == CHAIN_ID_TON) throw ERROR_INVALID_CHAIN_ID;
            val toAddress = buildAddress(0, comment.to);
            call(toAddress, 0, SEND_MODE_CARRY_ALL_REMAINING_MESSAGE_VALUE, (0x00000000 as uint32, comment.comment));
        }
        else => {
            assert (in.body.isEmpty()) throw ERROR_UNKNOWN_OPCODE;
        }
    }
}