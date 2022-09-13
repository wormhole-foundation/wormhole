#!/usr/bin/python3
"""
================================================================================================

The VAA Signature Verify Stateless Program

Copyright 2022 Wormhole Project Contributors

Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

------------------------------------------------------------------------------------------------

This program verifies a subset of the signatures in a VAA against the guardian set. This
program works in tandem with the VAA Processor stateful program.

The difference between this version and the Randlabs version is I removed most of the asserts
since we are going to have to completely validate the arguments again in the
TokenBridge contract.

We also cannot retroactively see/verify what arguments were passed into this
function unless all the arguments are in the Txn.application_args so
everything has to get moved out of the lsig args and into the txn_args

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
def sig_check(signatures, dhash, keys):
    """
    Verifies some signatures of a VAA. Due to computation budget limitations,
    this can't verify all signatures in one go. Instead, it just makes sure that
    whatever signatures it's given correspond to the given keys.

    In addition, none of the arguments are validated here beyond the fact that
    the signatures are valid given the keys and the message hash. In particular,
    the message hash is also not validated here. Thus, the proper way to use
    this function is by calling it (by the client) before the token bridge
    program.  Then the token bridge program verify each input + that the right
    program was called. If it failed to verify any of these, then signature
    verification could be bypaseed.
    """
    si = ScratchVar(TealType.uint64)  # signature index (zero-based)
    ki = ScratchVar(TealType.uint64)  # key index
    slen = ScratchVar(TealType.uint64)  # signature length
    rec_pk_x = ScratchVar(TealType.bytes, SLOTID_RECOVERED_PK_X)
    rec_pk_y = ScratchVar(TealType.bytes, SLOTID_RECOVERED_PK_Y)

    return Seq(
        [
            rec_pk_x.store(Bytes("")),
            rec_pk_y.store(Bytes("")),
            slen.store(Len(signatures)),
            For(Seq([
                si.store(Int(0)),
                ki.store(Int(0))
            ]),
                si.load() < slen.load(),
                Seq([
                    si.store(si.load() + Int(66)),
                    ki.store(ki.load() + Int(20))
                ])).Do(
                    Seq([
                        InlineAssembly(
                            "ecdsa_pk_recover Secp256k1",
                            dhash,
                            Btoi(Extract(signatures, si.load() + Int(65), Int(1))),
                            Extract(signatures, si.load() + Int(1), Int(32)),       # R
                            Extract(signatures, si.load() + Int(33), Int(32)),      # S
                            type=TealType.none),

                        # returned values in stack, pass to scratch-vars

                        InlineAssembly("store " + str(SLOTID_RECOVERED_PK_Y)),
                        InlineAssembly("store " + str(SLOTID_RECOVERED_PK_X)),

                        # Generate Ethereum-type public key, compare with guardian key.

                        Assert(Extract(keys, ki.load(), Int(20)) == Substring(Keccak256(Concat(rec_pk_x.load(), rec_pk_y.load())), Int(12), Int(32)))
                    ])
            ),
            Return(Int(1))
        ]
    )

def vaa_verify_program():
    signatures = Txn.application_args[1]
    keys = Txn.application_args[2]
    dhash = Txn.application_args[3]

    return Seq([
        Assert(Txn.rekey_to() == Global.zero_address()),
        Assert(Txn.fee() == Int(0)),
        Assert(Txn.type_enum() == TxnType.ApplicationCall),
        Assert(sig_check(signatures, dhash, keys)),
        Approve()]
    )

def get_vaa_verify(file_name = "teal/vaa_verify.teal"):
    teal = compileTeal(vaa_verify_program(), mode=Mode.Signature, version=6)

    with open(file_name, "w") as f:
        f.write(teal)

    return teal

if __name__ == '__main__':
    if len(sys.argv) == 2:
        get_vaa_verify(sys.argv[1])
    else:
        get_vaa_verify()
