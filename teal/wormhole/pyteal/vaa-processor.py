#!/usr/bin/python3
"""
================================================================================================

The VAA Processor Program

(c) 2021 Randlabs, Inc.

------------------------------------------------------------------------------------------------

This program is the core client to signed VAAs from Wormhole, working in tandem with the
verify-vaa.teal stateless programs.

The following application calls are available.

prepare:   Load globals in scatchspace to be used by stateless programs. Must be TX#0.
verify:  Verify guardian signature subset i..j, works in tandem with stateless program
commit:  Commit verified VAA, processing it according to its payload. Must be last TX.

------------------------------------------------------------------------------------------------

Global state:

"vphash"   :  Hash of verification program logic
"gsexp"    :  Guardian set expiration time
"gscount"  :  Guardian set size
key N      :  address of guardian N

------------------------------------------------------------------------------------------------
Stores in scratch: 

SLOT 254:  uint64 with bit field of approved guardians (initial 0)
SLOT 255:  number of guardians in set 
SLOT 4*i:  key of guardian i  (as of Nov'21 there are 19 guardians)
================================================================================================

"""
from pyteal import (compileTeal, Int, Mode, Txn, OnComplete, Itob, Btoi,
                    Return, Cond, Bytes, Global, Not, Seq, Approve, App, Assert, For, And)
import pyteal
from pyteal.ast.binaryexpr import ShiftRight
from pyteal.ast.if_ import If
from pyteal.ast.return_ import Reject
from pyteal.ast.scratch import ScratchLoad, ScratchSlot
from pyteal.ast.scratchvar import ScratchVar
from pyteal.ast.subroutine import Subroutine
from pyteal.ast.txn import TxnType
from pyteal.ast.while_ import While
from pyteal.types import TealType

METHOD = Txn.application_args[0]
SLOTID_SCRATCH_0 = 251
SLOTID_VERIFIED_GUARDIAN_BITS = 254
SLOTID_GUARDIAN_COUNT = 255

# Bootstrap with the initial list of guardians as application argument


@Subroutine(TealType.uint64)
def bootstrap():
    return Seq([
        App.globalPut(Bytes("vphash"), Txn.application_args[0]),
        Approve()
    ])


@Subroutine(TealType.uint64)
def is_creator():
    return Txn.sender() == Global.creator_address()


@Subroutine(TealType.uint64)
def is_proper_group_size():
    return Global.group_size() >= Int(3)


# @Subroutine(TealType.none)
# # set bitfield bits [from,to]
# def set_bits(i_from, i_to):
#     count = Int(1) #ScratchVar(TealType.uint64, SLOTID_SCRATCH_0)
#     #count.store(Int(i_to) - Int(i_from) + Int(1))
#     # set Verified_bits |= ((2^count) - 1) << i_from

#     bitfield = ScratchVar(TealType.uint64, SLOTID_VERIFIED_GUARDIAN_BITS)
#     return Seq([
#         bitfield.store(bitfield.load() | (
#             ((Int(2) ** count) - Int(1) << Int(2))
#         ))
#     ])


def prepare():
    # Sender must be owner
    # This call must be index 0 in a group of minimum 3 (prepare->verify->commit)

    ScratchVar(TealType.uint64, SLOTID_VERIFIED_GUARDIAN_BITS).store(Int(0))
    i = ScratchVar(TealType.uint64, SLOTID_SCRATCH_0)
    gs_count = ScratchVar(TealType.uint64, SLOTID_GUARDIAN_COUNT)
    num_guardians = App.globalGet(Bytes("gscount"))

    return Seq([
        Assert(And(is_creator(),
               is_proper_group_size(), Txn.group_index() == Int(0))),

        If(num_guardians == Int(0), Reject(), Seq([
            gs_count.store(num_guardians),
            For(i.store(Int(0)), i.load() < num_guardians, i.store(i.load() + Int(1))).Do(
                Seq([
                    ScratchVar(TealType.uint64, 1).store(
                        App.globalGet(Itob(i.load())))
                    # (ScratchVar(TealType.uint64, i.load())).store(App.globalGet(Itob(i.load()))
                ]))
        ])),
        Approve()
    ])


def verify():
    # Sender must be stateless logic.
    # This call must be not the first or last in a group of minimum 3 (prepare->verify->commit)

    # First guardian index to verify
    verify_from = Txn.application_args[0]
    # Last  guardian index to verify
    verify_to = Txn.application_args[1]
    bitfield = ScratchVar(TealType.uint64, SLOTID_VERIFIED_GUARDIAN_BITS)
    p_val = bitfield.load()
    count = Btoi(verify_to) - Btoi(verify_from) + Int(1)

    return Seq([Assert(And(is_proper_group_size(),
                       Txn.group_index() < Global.group_size(),
                       Txn.group_index() > Int(0),
                       Btoi(verify_to) > Btoi(verify_from),
                       Txn.sender() == App.globalGet(Bytes("vphash")))),
                #bitfield.store(bitfield.load() + Int(1)),
                Approve()])


def commit():
    # Sender must be owner
    # This call must be last in a group of minimum 3 (prepare->verify->commit)
    # Bitfield must indicate all guardians verified.
    all_verified = ScratchVar(TealType.uint64, SLOTID_VERIFIED_GUARDIAN_BITS).load() ==
    return Seq([
        Assert(And(is_proper_group_size(),
                   Txn.group_index() == (Global.group_size() - Int(1)),
                   all_verified
                   )),
        handle_vaa(),
        Approve()
    ])


def vaa_processor_program():
    handle_create = Return(bootstrap())
    handle_update = Return(is_creator())
    handle_delete = Return(is_creator())
    handle_noop = Cond(
        [METHOD == Bytes("prepare"), prepare()],
        [METHOD == Bytes("verify"), verify()],
        [METHOD == Bytes("commit"), commit()]
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
    with open("vaa-processor-approval.teal", "w") as f:
        compiled = compileTeal(vaa_processor_program(),
                               mode=Mode.Application, version=5)
        f.write(compiled)

    with open("vaa-processor-clear.teal", "w") as f:
        compiled = compileTeal(clear_state_program(),
                               mode=Mode.Application, version=5)
        f.write(compiled)
