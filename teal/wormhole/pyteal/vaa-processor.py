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
from pyteal import (compileTeal, Int, Mode, Txn, OnComplete,
                    Return, Cond, Bytes, Global, Not, Seq, Approve, App, Assert, For, And)
import pyteal
from pyteal.ast.binaryexpr import ShiftRight
from pyteal.ast.if_ import If
from pyteal.ast.return_ import Reject
from pyteal.ast.scratch import ScratchLoad, ScratchSlot
from pyteal.ast.scratchvar import ScratchVar
from pyteal.ast.subroutine import Subroutine
from pyteal.ast.while_ import While
from pyteal.types import TealType

METHOD = Txn.application_args[0]

# Bootstrap with the initial list of guardians as application argument


@Subroutine(TealType.uint64)
def bootstrap():
    return Seq([
        Approve()
    ])


@Subroutine(TealType.uint64)
def is_creator():
    return Txn.sender() == Global.creator_address()


@Subroutine(TealType.uint64)
def is_proper_group_size():
    return Global.group_size() >= Int(3)


def prepare():
    # Sender must be owner
    # This call must be index 0 in a group of minimum 3 (prepare->verify->commit)
    Assert(And(is_creator(),
               is_proper_group_size(), Txn.group_index() == Int(0)))

    i = ScratchVar(TealType.uint64, 253)
    gs_count = ScratchVar(TealType.uint64, 254)
    num_guardians = App.globalGet(Bytes("gscount"))
    If(num_guardians == Int(0), Reject(), Seq([
        gs_count.store(num_guardians),
        For(i.store(Int(0)), i.load() < num_guardians, i.store(i.load() + Int(1))).Do(
            Seq([
                
                # (ScratchVar(TealType.uint64, i.load())).store(App.globalGet(Itob(i.load()))
            ]))
    ]))
    return Approve()


def verify():
    # This call must be not the first or last in a group of minimum 3 (prepare->verify->commit)
    return Seq([Assert(And(is_proper_group_size(),
                       Txn.group_index() < Global.group_size(),
                       Txn.group_index() > Int(0))),
                Approve()])


def commit():
    # Sender must be owner
    # This call must be last in a group of minimum 3 (prepare->verify->commit)
    return Seq([
        Assert(And(is_proper_group_size(), Txn.group_index()
               == (Global.group_size() - Int(1)))),
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
        [Not(Txn.application_id()), handle_create],
        [Txn.on_completion() == OnComplete.UpdateApplication, handle_update],
        [Txn.on_completion() == OnComplete.DeleteApplication, handle_delete],
        [Txn.on_completion() == OnComplete.NoOp, handle_noop]
    )


def clear_state_program():
    Return(Int(1))


if __name__ == "__main__":
    print(compileTeal(vaa_processor_program(), mode=Mode.Application, version=5))
