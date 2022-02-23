#!/usr/bin/python3
"""
================================================================================================

The Pricekeeper II Program

v3.0

(c) 2022 Wormhole Project Contributors

------------------------------------------------------------------------------------------------
v1.0 - first version
v2.0 - stores Pyth Payload
v3.0 - supports Pyth V2 "batched" price Payloads.

This program stores price data verified from Pyth VAA messaging. To accept data, this application
requires to be the last of the verification transaction group, and the verification condition
bits must be set.

The following application calls are available.

submit: Submit payload.  

The payload format must be V2, with batched message support.

------------------------------------------------------------------------------------------------

Global state:

key             Concatenated productId + priceId
value           packed fields as follow: 

                Bytes
                
                8               price
                4               exponent
                8               twap value
                8               twac value
                8               confidence
                1               status
                1               corporate act
                8               timestamp (based on Solana contract call time)
                ------------------------------
                Total: 109 bytes.

------------------------------------------------------------------------------------------------
"""
from pyteal.ast import *
from pyteal.types import *
from pyteal.compiler import *
from pyteal.ir import *
from globals import *
import sys

METHOD = Txn.application_args[0]
PYTH_PAYLOAD = Txn.application_args[1]
SLOTID_VERIFIED_BIT = 254
SLOT_VERIFIED_BITFIELD = ScratchVar(TealType.uint64, SLOTID_VERIFIED_BIT)
SLOT_TEMP = ScratchVar(TealType.uint64)
VAA_PROCESSOR_APPID = App.globalGet(Bytes("vaapid"))
PYTH_ATTESTATION_V2_BYTES = 150


@Subroutine(TealType.uint64)
def is_creator():
    return Txn.sender() == Global.creator_address()


@Subroutine(TealType.uint64)
# Arg0: Bootstrap with the authorized VAA Processor appid.
def bootstrap():
    return Seq([
        App.globalPut(Bytes("vaapid"), Btoi(Txn.application_args[0])),
        Approve()
    ])


@Subroutine(TealType.uint64)
def check_group_tx():
    #
    # Verifies that previous steps had set their verification bits.
    # Verifies that previous steps are app calls issued from authorized appId.
    #
    i = SLOT_TEMP
    return Seq([
        For(i.store(Int(1)),
            i.load() < Global.group_size() - Int(1),
            i.store(i.load() + Int(1))).Do(Seq([
                Assert(Gtxn[i.load()].type_enum() == TxnType.ApplicationCall),
                Assert(Gtxn[i.load()].application_id()
                       == VAA_PROCESSOR_APPID),
                Assert(GetBit(ImportScratchValue(i.load() - Int(1),
                       SLOTID_VERIFIED_BIT), i.load() - Int(1)) == Int(1))
            ])
        ),
        Return(Int(1))
    ])


def store():
    # * Sender must be owner
    # * This must be part of a transaction group
    # * All calls in group must be issued from authorized appid.
    # * All calls in group must have verification bits set.
    # * Argument 0 must be Pyth payload.

    pyth_payload = ScratchVar(TealType.bytes)
    packed_price_data = ScratchVar(TealType.bytes)
    num_attestations = ScratchVar(TealType.uint64)
    attestation_size = ScratchVar(TealType.uint64)
    attestation_data = ScratchVar(TealType.bytes)
    product_price_key = ScratchVar(TealType.bytes)

    i = ScratchVar(TealType.uint64)

    return Seq([
        pyth_payload.store(PYTH_PAYLOAD),
        Assert(Global.group_size() > Int(1)),
        Assert(Txn.application_args.length() == Int(2)),
        Assert(is_creator()),
        Assert(check_group_tx()),
        
        # check magic header and version
        Assert(Extract(pyth_payload.load(), Int(0), Int(4)) == Bytes("\x50\x32\x57\x48")),
        Assert(Extract(pyth_payload.load(), Int(4), Int(2)) == Bytes("\x00\x02")),
        
        # get attestation count
        num_attestations.store(Btoi(Extract(pyth_payload.load(), Int(7), Int(2)))),
        Assert(num_attestations.load() > Int(0)),

        # ensure standard V2 format 150-byte attestation
        attestation_size.store(Btoi(Extract(pyth_payload.load(), Int(9), Int(2)))),
        Assert(attestation_size.load() == Int(PYTH_ATTESTATION_V2_BYTES)),
        
        # this message size must agree with data in fields
        Assert(attestation_size.load() * num_attestations.load() + Int(11) == Len(pyth_payload.load())),
        
        # Read each attestation, store in global state.

        For(i.store(Int(0)), i.load() < num_attestations.load(), i.store(i.load() + Int(1))).Do(
            Seq([
                attestation_data.store(Extract(pyth_payload.load(), Int(11) + (Int(PYTH_ATTESTATION_V2_BYTES) * i.load()), Int(PYTH_ATTESTATION_V2_BYTES))),
                product_price_key.store(Extract(attestation_data.load(), Int(7), Int(64))),
                packed_price_data.store(Concat(
                    Extract(attestation_data.load(), Int(72), Int(20)),   # price + exponent + twap
                    Extract(attestation_data.load(), Int(100), Int(8)),  # store twac
                    Extract(attestation_data.load(), Int(132), Int(18)),  # confidence, status, corpact, timestamp
                )),
                App.globalPut(product_price_key.load(), packed_price_data.load()),
                ])
        ),
        Approve()])


def pricekeeper_program():
    handle_create = Return(bootstrap())
    handle_update = Return(is_creator())
    handle_delete = Return(is_creator())
    handle_noop = Cond(
        [METHOD == Bytes("store"), store()],
    )
    return Cond(
        [Txn.application_id() == Int(0), handle_create],
        [Txn.on_completion() == OnComplete.UpdateApplication, handle_update],
        [Txn.on_completion() == OnComplete.DeleteApplication, handle_delete],
        [Txn.on_completion() == OnComplete.NoOp, handle_noop]
    )


def clear_state_program():
    return Int(1)


if __name__ == "__main__":

    approval_outfile = "teal/wormhole/build/pricekeeper-v2-approval.teal"
    clear_state_outfile = "teal/wormhole/build/pricekeeper-v2-clear.teal"

    if len(sys.argv) >= 2:
        approval_outfile = sys.argv[1]

    if len(sys.argv) >= 3:
        clear_state_outfile = sys.argv[2]

    print("Pricekeeper V2 Program, (c) 2022 Wormhole Project Contributors")
    print("Compiling approval program...")

    with open(approval_outfile, "w") as f:
        compiled = compileTeal(pricekeeper_program(),
                               mode=Mode.Application, version=5)
        f.write(compiled)

    print("Written to " + approval_outfile)
    print("Compiling clear state program...")

    with open(clear_state_outfile, "w") as f:
        compiled = compileTeal(clear_state_program(),
                               mode=Mode.Application, version=5)
        f.write(compiled)

    print("Written to " + clear_state_outfile)
