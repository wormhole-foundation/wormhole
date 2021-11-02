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
from pyteal.ast import *
from pyteal.types import *
from pyteal.compiler import *
from pyteal.ir import *
from globals import get_sig_count_in_step, get_group_size


@Subroutine(TealType.uint64)
def sig_check(signatures, digest, keys):
    si = ScratchVar(TealType.uint64)
    ki = ScratchVar(TealType.uint64)
    return Seq(
        [
            For(Seq([
                si.store(Int(0)),
                ki.store(Int(0))
            ]),
                si.load() < Len(signatures),
                Seq([
                    si.store(si.load() + Int(66)),
                    ki.store(ki.load() + Int(32)),
                ])).Do(
                Seq(
                    Assert(Ed25519Verify(
                        digest,
                        Extract(signatures, si.load(), Int(66)),
                        Extract(keys, ki.load(), Int(32)),))
                )
            ),
            Return(Int(1))
        ]
    )


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
    num_guardians = Txn.application_args[2]

    return Seq([
        Assert(And(
            Txn.fee() <= Int(1000),
            Txn.application_args.length() == Int(1),
            Len(signatures) == get_sig_count_in_step(
                Txn.group_index(), Btoi(num_guardians)) * Int(66),
            Txn.rekey_to() == Global.zero_address(),
            Txn.application_id() == Int(vaa_processor_app_id),
            Txn.type_enum() == TxnType.ApplicationCall,
            Global.group_size() == get_group_size(Btoi(num_guardians)),
            sig_check(signatures, digest, keys))
        ),
        Approve()])


if __name__ == "__main__":
    with open("teal/wormhole/build/vaa-verify.teal", "w") as f:
        compiled = compileTeal(vaa_verify_program(
            333), mode=Mode.Signature, version=5)
        f.write(compiled)
