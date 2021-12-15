#!/usr/bin/python3
"""
================================================================================================

The Pricekeeper II Program

(c) 2021-22 Randlabs, Inc.

------------------------------------------------------------------------------------------------

This program stores price data verified from Pyth VAA messaging. To accept data, this application
requires to be the last of the verification transaction group, and the verification condition
bits must be set.

The following application calls are available.

submit: Submit payload.  
This must be 150-bytes long (Pyth native payload.)
------------------------------------------------------------------------------------------------

Global state:

key             name of symbol
value           packed fields as follow: 

                Bytes
                32              productId
                32              priceId
                8               price
                1               price_type
                4               exponent
                8               twap value
                8               twac value
                8               confidence
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
ARG_SYMBOL_NAME = Txn.application_args[1]
ARG_PRICE_DATA = Txn.application_args[2]
SLOTID_VERIFIED_BIT = 254
SLOT_VERIFIED_BITFIELD = ScratchVar(TealType.uint64, SLOTID_VERIFIED_BIT)
SLOT_TEMP = ScratchVar(TealType.uint64)
VAA_PROCESSOR_APPID = App.globalGet(Bytes("vaapid"))
PYTH_PAYLOAD_LENGTH_BYTES = 150


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
    # * Argument 0 must be price symbol name.
    # * Argument 1 must be Pyth payload (150 bytes long)

    pyth_price_data = ScratchVar(TealType.bytes)
    packed_price_data = ScratchVar(TealType.bytes)
    return Seq([
        pyth_price_data.store(ARG_PRICE_DATA),
        Assert(Global.group_size() > Int(1)),
        Assert(Len(pyth_price_data.load()) == Int(PYTH_PAYLOAD_LENGTH_BYTES)),
        Assert(Txn.application_args.length() == Int(3)),
        Assert(is_creator()),
        Assert(check_group_tx()),

        # Unpack Pyth payload and store the data we want (see doc at beginning)

        packed_price_data.store(Concat(
            # store product_id, price_id, price_type, price, exponent, twap
            Extract(pyth_price_data.load(), Int(14), Int(85)),
            Extract(pyth_price_data.load(), Int(108), Int(8)),  # store twac
            Extract(pyth_price_data.load(), Int(132), Int(8)),  # confidence
            Extract(pyth_price_data.load(), Int(142), Int(8)),  # timestamp
        )),
        App.globalPut(ARG_SYMBOL_NAME, packed_price_data.load()),
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

    print("Pricekeeper V2 Program, (c) 2021-22 Randlabs Inc. ")
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
