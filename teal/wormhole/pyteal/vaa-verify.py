#!/usr/bin/python3
"""
================================================================================================

The VAA Signature Verify Stateless Program

(c) 2021 Randlabs, Inc.

------------------------------------------------------------------------------------------------

This program verifies a subset of the signatures in a VAA against the guardian set. This
program works in tandem with the VAA Processor stateful program.

================================================================================================

"""
from pyteal import (compileTeal, Int, Mode, Txn, OnComplete, Itob, Btoi,
                    ImportScratchValue,
                    Return, Cond, Bytes, Global, Not, Gtxn, Seq, Approve, App, Assert, For, Len, And)
from pyteal.ast import Subroutine
from pyteal.ast.arg import Arg
from pyteal.ast.txn import TxnType
from pyteal.types import TealType

from globals import SIGNATURES_PER_VERIFICATION_STEP, is_proper_group_size

"""
* Let N be the number of signatures per verification step, for the TX(i) in group, we verify signatures [j..k] where j = i*N, k = j+(N-1)
* Input 0 is signatures [j..k] to verify as LogicSigArgs.
* Input 1 is signed digest of payload, contained in the note field of the TX in current slot.
* Input 2 is public keys for guardians [j..k] contained in the first Argument  of the TX in current slot.
* Input 3 is guardian set size contained in the second argument of the TX in current slot.
"""


def vaa_verify_program(vaa_processor_app_id):
    signatures = Arg(0)
    digest = Txn.note()
    keys = Txn.application_args[1]
    gssize = Txn.application_args[2]

    return Seq([
        Assert(And(
            Txn.fee() <= Int(1000),
            Txn.application_args.length() == Int(1),
            Len(signatures) == Int(SIGNATURES_PER_VERIFICATION_STEP) * Int(66),
            Txn.rekey_to() == Global.zero_address(),
            Txn.application_id() == Int(vaa_processor_app_id),
            Txn.type_enum() == TxnType.ApplicationCall,
            is_proper_group_size(Btoi(gssize))),),
        Approve()])


if __name__ == "__main__":
    with open("vaa-verify.teal", "w") as f:
        compiled = compileTeal(vaa_verify_program(
            333), mode=Mode.Signature, version=5)
        f.write(compiled)
