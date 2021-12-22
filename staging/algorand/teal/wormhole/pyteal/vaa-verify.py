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
from globals import *
from inlineasm import *

import sys

SLOTID_RECOVERED_PK_X = 240
SLOTID_RECOVERED_PK_Y = 241


@Subroutine(TealType.uint64)
def sig_check(signatures, digest, keys):
    si = ScratchVar(TealType.uint64)
    ki = ScratchVar(TealType.uint64)
    i = ScratchVar(TealType.uint64)
    rec_pk_x = ScratchVar(TealType.bytes, SLOTID_RECOVERED_PK_X)
    rec_pk_y = ScratchVar(TealType.bytes, SLOTID_RECOVERED_PK_Y)

    return Seq(
        [
            rec_pk_x.store(Bytes("")),
            rec_pk_y.store(Bytes("")),
            For(Seq([
                i.store(Int(0)),
                si.store(Int(0)),
                ki.store(Int(0))
            ]),
                si.load() < Len(signatures),
                Seq([
                    si.store(si.load() + Int(66)),
                    ki.store(ki.load() + Int(20)),
                    i.store(i.load() + Int(1)),
                ])).Do(
                    Seq([
                        # Index must be sequential

                        Assert(Btoi(Extract(signatures, si.load(), Int(1))) == 
                             i.load() + (Txn.group_index() * Int(MAX_SIGNATURES_PER_VERIFICATION_STEP))),

                        InlineAssembly(
                            "ecdsa_pk_recover Secp256k1",
                            Keccak256(digest),
                            Btoi(Extract(signatures, si.load() + Int(65), Int(1))),
                            Extract(signatures, si.load() + Int(1), Int(32)),       # R
                            Extract(signatures, si.load() + Int(33), Int(32)),      # S
                            type=TealType.none),

                        # returned values in stack, pass to scratch-vars

                        InlineAssembly("store " + str(SLOTID_RECOVERED_PK_Y)),
                        InlineAssembly("store " + str(SLOTID_RECOVERED_PK_X)),

                        # Generate Ethereum-type public key, compare with guardian key.

                        Assert(
                            Extract(keys, ki.load(), Int(20)) ==
                            Substring(Keccak256(Concat(rec_pk_x.load(),
                                    rec_pk_y.load())), Int(12), Int(32))
                        )
                    ])


            ),
            Return(Int(1))
        ]
    )


"""
* Let N be the number of signatures per verification step, for the TX(i) in group, we verify signatures [j..k] where j = i*N, k = j+(N-1)
* Input 0 is signatures [j..k] to verify as LogicSigArgs. (Format is GuardianIndex + signature)
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
        Assert(Txn.fee() <= Int(1000)),
        Assert(Txn.application_args.length() == Int(3)),
        Assert(Len(signatures) == get_sig_count_in_step(
                Txn.group_index(), Btoi(num_guardians)) * Int(66)),
        Assert(Txn.rekey_to() == Global.zero_address()),
        Assert(Txn.application_id() == Int(vaa_processor_app_id)),
        Assert(Txn.type_enum() == TxnType.ApplicationCall),
        Assert(Global.group_size() == get_group_size(Btoi(num_guardians))),
        Assert(sig_check(signatures, digest, keys)),
        Approve()]
    )

if __name__ == "__main__":
    outfile = "teal/wormhole/build/vaa-verify.teal"
    appid = 0

    print("VAA Verify Stateless Program, (c) 2021-22 Randlabs Inc. ")
    print("Compiling...")

    if len(sys.argv) >= 1:
        appid = sys.argv[1]

    if len(sys.argv) >= 2:
        outfile = sys.argv[2]

    with open(outfile, "w") as f:
        compiled = compileTeal(vaa_verify_program(
            int(appid)), mode=Mode.Signature, version=5)
        f.write(compiled)

    print("Written to " + outfile)
