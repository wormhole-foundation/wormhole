from time import time, sleep
from typing import List, Tuple, Dict, Any, Optional, Union
from base64 import b64decode
import base64
import random
import hashlib
import uuid
import sys
import json
import uvarint

from local_blob import LocalBlob
from TmplSig import TmplSig

from algosdk.v2client.algod import AlgodClient
from algosdk.kmd import KMDClient
from algosdk import account, mnemonic
from algosdk.encoding import decode_address
from algosdk.future import transaction
from pyteal import compileTeal, Mode, Expr
from pyteal import *
from algosdk.logic import get_application_address

from algosdk.future.transaction import LogicSigAccount
from inspect import currentframe

import pprint

max_keys = 15
max_bytes_per_key = 127
bits_per_byte = 8

bits_per_key = max_bytes_per_key * bits_per_byte
max_bytes = max_bytes_per_key * max_keys
max_bits = bits_per_byte * max_bytes

def fullyCompileContract(genTeal, client: AlgodClient, contract: Expr, name, devmode) -> bytes:
    if genTeal:
        if devmode:
            teal = compileTeal(contract, mode=Mode.Application, version=6, assembleConstants=True)
        else:
            teal = compileTeal(contract, mode=Mode.Application, version=6, assembleConstants=True, optimize=OptimizeOptions(scratch_slots=True))

        with open(name, "w") as f:
            print("Writing " + name)
            f.write(teal)
    else:
        with open(name, "r") as f:
            print("Reading " + name)
            teal = f.read()

    response = client.compile(teal)

    with open(name + ".bin", "w") as fout:
        fout.write(response["result"])
    with open(name + ".hash", "w") as fout:
        fout.write(decode_address(response["hash"]).hex())

    return response

def getCoreContracts(   genTeal, approve_name, clear_name,
                        client: AlgodClient,
                        seed_amt: int,
                        tmpl_sig: TmplSig,
                        devMode: bool
                        ) -> Tuple[bytes, bytes]:

    def vaa_processor_program(seed_amt: int, tmpl_sig: TmplSig):
        blob = LocalBlob()

        def MagicAssert(a) -> Expr:
            if devMode:
                return Assert(And(a, Int(currentframe().f_back.f_lineno)))
            else:
                return Assert(a)

        @Subroutine(TealType.bytes)
        def encode_uvarint(val: Expr, b: Expr):
            buff = ScratchVar()
            return Seq(
                buff.store(b),
                Concat(
                    buff.load(),
                    If(
                        val >= Int(128),
                        encode_uvarint(
                            val >> Int(7),
                            Extract(Itob((val & Int(255)) | Int(128)), Int(7), Int(1)),
                        ),
                        Extract(Itob(val & Int(255)), Int(7), Int(1)),
                    ),
                ),
            )

        @Subroutine(TealType.bytes)
        def get_sig_address(acct_seq_start: Expr, emitter: Expr):
            # We could iterate over N items and encode them for a more general interface
            # but we inline them directly here
    
            return Sha512_256(
                Concat(
                Bytes("Program"),
                # ADDR_IDX aka sequence start
                tmpl_sig.get_bytecode_chunk(0),
                encode_uvarint(acct_seq_start, Bytes("")),

                # EMITTER_ID
                tmpl_sig.get_bytecode_chunk(1),
                encode_uvarint(Len(emitter), Bytes("")),
                emitter,

                # APP_ID
                tmpl_sig.get_bytecode_chunk(2),
                encode_uvarint(Global.current_application_id(), Bytes("")),

                # TMPL_APP_ADDRESS
                tmpl_sig.get_bytecode_chunk(3),
                encode_uvarint(Len(Global.current_application_address()), Bytes("")),
                Global.current_application_address(),


                tmpl_sig.get_bytecode_chunk(4),
                )
            )
    
        @Subroutine(TealType.uint64)
        def optin():
            # Alias for readability
            algo_seed = Gtxn[Txn.group_index() - Int(1)]
            optin = Txn
    
            well_formed_optin = And(
                # Check that we're paying it
                algo_seed.type_enum() == TxnType.Payment,
                algo_seed.amount() == Int(seed_amt),
                algo_seed.receiver() == optin.sender(),
                # Check that its an opt in to us
                optin.type_enum() == TxnType.ApplicationCall,
                optin.on_completion() == OnComplete.OptIn,
                # Not strictly necessary since we wouldn't be seeing this unless it was us, but...
                optin.application_id() == Global.current_application_id(),
                optin.rekey_to() == Global.current_application_address(),
                optin.application_args.length() == Int(0)
            )
    
            return Seq(
                # Make sure its a valid optin
                MagicAssert(well_formed_optin),
                # Init by writing to the full space available for the sender (Int(0))
                blob.zero(Int(0)),
                # we gucci
                Int(1)
            )
    
        def nop():
            return Seq([Approve()])

        def publishMessage():
            seq = ScratchVar()
            fee = ScratchVar()

            pmt = Gtxn[Txn.group_index() - Int(1)]

            return Seq([
                # Lets see if we were handed the correct account to store the sequence number in
                MagicAssert(Txn.accounts[1] == get_sig_address(Int(0), Txn.sender())),

                fee.store(App.globalGet(Bytes("MessageFee"))),
                If(fee.load() > Int(0), Seq([
                        MagicAssert(And(
                            pmt.type_enum() == TxnType.Payment,
                            pmt.amount() >= fee.load(),
                            pmt.receiver() == Global.current_application_address(),
                            pmt.rekey_to() == Global.zero_address()
                        )),
                ])),

                # emitter sequence number
                seq.store(Itob(Btoi(blob.read(Int(1), Int(0), Int(8))) + Int(1))),
                Pop(blob.write(Int(1), Int(0), seq.load())),

                # Log it so that we can look for this on the guardian network
                Log(seq.load()),

                blob.meta(Int(1), Bytes("publishMessage")),
                
                Approve()
            ])

        def hdlGovernance(isBoot: Expr):
            off = ScratchVar()
            a = ScratchVar()
            emitter = ScratchVar()
            dest = ScratchVar()
            fee = ScratchVar()
            idx = ScratchVar()
            set = ScratchVar()
            len = ScratchVar()
            v = ScratchVar()
            tchain = ScratchVar()

            return Seq([

                # All governance must be done with the most recent guardian set
                set.store(App.globalGet(Bytes("currentGuardianSetIndex"))),
                If(set.load() != Int(0), Seq([
                        idx.store(Extract(Txn.application_args[1], Int(1), Int(4))),
                        MagicAssert(Btoi(idx.load()) == set.load()),
                ])),

                # The offset of the chain
                off.store(Btoi(Extract(Txn.application_args[1], Int(5), Int(1))) * Int(66) + Int(14)), 
                # Correct source chain? 
                MagicAssert(Extract(Txn.application_args[1], off.load(), Int(2)) == Bytes("base16", "0001")),
                # Correct emitter?
                MagicAssert(Extract(Txn.application_args[1], off.load() + Int(2), Int(32)) == Bytes("base16", "0000000000000000000000000000000000000000000000000000000000000004")),
                # Get us to the payload
                off.store(off.load() + Int(43)),
                # Is this a governance message?
                MagicAssert(Extract(Txn.application_args[1], off.load(), Int(32)) == Bytes("base16", "00000000000000000000000000000000000000000000000000000000436f7265")),
                off.store(off.load() + Int(32)),
                # What is the target of this governance message?
                tchain.store(Extract(Txn.application_args[1], off.load() + Int(1), Int(2))),
                # Needs to point at us or to all chains

                a.store(Btoi(Extract(Txn.application_args[1], off.load(), Int(1)))),
                Cond( 
                    [a.load() == Int(1), Seq([
                        # ContractUpgrade is a VAA that instructs an implementation on a specific chain to upgrade itself
                        # 
                        # In the case of Algorand, it contains the hash of the program that we are allowed to upgrade ourselves to.  We would then run the upgrade program itself
                        # to perform the actual upgrade
                        MagicAssert(tchain.load() == Bytes("base16", "0008")),
                        
                        off.store(off.load() + Int(3)),

                        App.globalPut(Bytes("validUpdateApproveHash"), Extract(Txn.application_args[1], off.load(), Int(32)))
                    ])],
                    [a.load() == Int(2), Seq([
                        # We are updating the guardian set

                        # This should point at all chains

                        MagicAssert(Or(tchain.load() == Bytes("base16", "0008"), tchain.load() == Bytes("base16", "0000"))),

                        # move off to point at the NewGuardianSetIndex and grab it
                        off.store(off.load() + Int(3)),
                        v.store(Extract(Txn.application_args[1], off.load(), Int(4))),
                        idx.store(Btoi(v.load())),

                        # Lets see if the user handed us the correct memory... no hacky hacky
                        MagicAssert(Txn.accounts[3] == get_sig_address(idx.load(), Bytes("guardian"))), 

                        # Make sure it is different and we can only walk forward
                        If(isBoot == Int(0), Seq(
                                MagicAssert(Txn.accounts[3] != Txn.accounts[2]),
                                MagicAssert(idx.load() == (set.load() + Int(1)))
                        )),

                        # Write this away till the next time
                        App.globalPut(Bytes("currentGuardianSetIndex"), idx.load()),

                        # Write everything out to the auxilliary storage
                        off.store(off.load() + Int(4)),
                        len.store(Btoi(Extract(Txn.application_args[1], off.load(), Int(1)))),

                        # Lets not let us get bricked by somebody submitting a stupid guardian set...
                        MagicAssert(len.load() > Int(0)),  

                        Pop(blob.write(Int(3), Int(0), Extract(Txn.application_args[1], off.load(), Int(1) + (Int(20) * len.load())))),

                        If(Txn.accounts[3] != Txn.accounts[2],
                           Pop(blob.write(Int(2), Int(1000), Itob(Global.latest_timestamp() + Int(86400))))),
                        blob.meta(Int(3), Bytes("guardian"))
                    ])],
                    [a.load() == Int(3), Seq([
                        off.store(off.load() + Int(1)),
                        MagicAssert(tchain.load() == Bytes("base16", "0008")),
                        off.store(off.load() + Int(2) + Int(24)),
                        fee.store(Btoi(Extract(Txn.application_args[1], off.load(), Int(8)))),
                        App.globalPut(Bytes("MessageFee"), fee.load()),
                    ])],
                    [a.load() == Int(4), Seq([
                        off.store(off.load() + Int(1)),
                        MagicAssert(tchain.load() == Bytes("base16", "0008")),
                        off.store(off.load() + Int(26)),
                        fee.store(Btoi(Extract(Txn.application_args[1], off.load(), Int(8)))),
                        off.store(off.load() + Int(8)),
                        dest.store(Extract(Txn.application_args[1], off.load(), Int(32))),

                        InnerTxnBuilder.Begin(),
                        InnerTxnBuilder.SetFields(
                              {
                                  TxnField.type_enum: TxnType.Payment,
                                  TxnField.receiver: dest.load(),
                                  TxnField.amount: fee.load(),
                                  TxnField.fee: Int(0),
                              }
                        ),
                        InnerTxnBuilder.Submit(),
                    ])]
               ),
                Approve()
            ])

        def init():
            return Seq([
                # You better lose yourself in the music, the moment
                App.globalPut(Bytes("vphash"), Txn.application_args[2]),

                # You own it, you better never let it go
                MagicAssert(Txn.sender() == Global.creator_address()),

                # You only get one shot, do not miss your chance to blow
                MagicAssert(App.globalGet(Bytes("booted")) == Int(0)),
                App.globalPut(Bytes("booted"), Bytes("true")),

                # This opportunity comes once in a lifetime
                checkForDuplicate(),

                # You can do anything you set your mind to...
                hdlGovernance(Int(1))
            ])

        def verifySigs():
            return Return (Txn.sender() == STATELESS_LOGIC_HASH)

        @Subroutine(TealType.none)
        def checkForDuplicate():
            off = ScratchVar()
            emitter = ScratchVar()
            sequence = ScratchVar()
            b = ScratchVar()
            byte_offset = ScratchVar()

            return Seq(
                # VM only is version 1
                MagicAssert(Btoi(Extract(Txn.application_args[1], Int(0), Int(1))) == Int(1)),

                off.store(Btoi(Extract(Txn.application_args[1], Int(5), Int(1))) * Int(66) + Int(14)), # The offset of the emitter

                # emitter is chain/contract-address
                emitter.store(Extract(Txn.application_args[1], off.load(), Int(34))),
                sequence.store(Btoi(Extract(Txn.application_args[1], off.load() + Int(34), Int(8)))),

                # They passed us the correct account?  In this case, byte_offset points at the whole block
                byte_offset.store(sequence.load() / Int(max_bits)),
                MagicAssert(Txn.accounts[1] == get_sig_address(byte_offset.load(), emitter.load())),

                # Now, lets go grab the raw byte
                byte_offset.store((sequence.load() / Int(8)) % Int(max_bytes)),
                b.store(blob.get_byte(Int(1), byte_offset.load())),

                # I would hope we've never seen this packet before...   throw an exception if we have
                MagicAssert(GetBit(b.load(), sequence.load() % Int(8)) == Int(0)),

                # Lets mark this bit so that we never see it again
                blob.set_byte(Int(1), byte_offset.load(), SetBit(b.load(), sequence.load() % Int(8), Int(1))),

                blob.meta(Int(1), Bytes("duplicate"))
            )

        STATELESS_LOGIC_HASH = App.globalGet(Bytes("vphash"))

        def verifyVAA():
            i = ScratchVar()
            a = ScratchVar()
            total_guardians = ScratchVar()
            guardian_keys = ScratchVar()
            num_sigs = ScratchVar()
            off = ScratchVar()
            digest = ScratchVar()
            hits = ScratchVar()
            s = ScratchVar()
            eoff = ScratchVar()
            guardian = ScratchVar()

            return Seq([
                # We have a guardian set?  We have OUR guardian set?
                MagicAssert(Txn.accounts[2] == get_sig_address(Btoi(Extract(Txn.application_args[1], Int(1), Int(4))), Bytes("guardian"))),
                blob.checkMeta(Int(2), Bytes("guardian")),
                # Lets grab the total keyset
                total_guardians.store(blob.get_byte(Int(2), Int(0))),
                MagicAssert(total_guardians.load() > Int(0)),

                guardian_keys.store(blob.read(Int(2), Int(1), Int(1) + Int(20) * total_guardians.load())),

                # I wonder if this is an expired guardian set
                s.store(Btoi(blob.read(Int(2), Int(1000), Int(1008)))),
                If(s.load() != Int(0),
                   MagicAssert(Global.latest_timestamp() < s.load())),

                hits.store(Bytes("base16", "0x00000000")),

                # How many signatures are in this vaa?
                num_sigs.store(Btoi(Extract(Txn.application_args[1], Int(5), Int(1)))),

                # Lets create a digest of THIS vaa...
                off.store(Int(6) + (num_sigs.load() * Int(66))),
                digest.store(Keccak256(Keccak256(Extract(Txn.application_args[1], off.load(), Len(Txn.application_args[1]) - off.load())))),

                # We have enough signatures?
                MagicAssert(And(
                    total_guardians.load() > Int(0),
                    num_sigs.load() <= total_guardians.load(),
                    num_sigs.load() > ((total_guardians.load() * Int(2)) / Int(3)),
                    )),


                # Point it at the start of the signatures in the VAA
                off.store(Int(6)),

                # We'll check that the preceding transactions properly verify
                # all of the signatures. Due to size limitations, there will be
                # multiple 'verifySigs' calls to achieve this. First we walk
                # backwards from the current instruction to find all the
                # 'verifySigs' calls. We do it this way because it's possible
                # that the VAA transactions are composed with some other
                # contracts calls, so we do not rely in absolute transaction
                # indices.
                #
                # | | ...            |
                # | | something else |
                # | |----------------|
                # | | verifySigs     |
                # | | verifySigs     |
                # | | verifySigs     |
                # | | verifyVAA      | <- we are here now
                # | |----------------|
                # v | ...            |

                MagicAssert(Txn.group_index() > Int(0)),
                # the first 'verifySigs' tx is the one before us
                i.store(Txn.group_index() - Int(1)),
                MagicAssert(Gtxn[i.load()].application_args.length() > Int(0)),
                a.store(Gtxn[i.load()].application_args[0]),

                # Go back until we hit 'something else' or run out of
                # transactions (we allow nops too)
                While (And(i.load() > Int(0), Or(a.load() == Bytes("verifySigs"), a.load() == Bytes("nop")))).Do(Seq([
                        i.store(i.load() - Int(1)),
                        If (Gtxn[i.load()].application_args.length() > Int(0),
                            a.store(Gtxn[i.load()].application_args[0]),
                            Seq([
                                a.store(Bytes("")),
                                Break()
                            ]))
                ])),

                If(And(a.load() != Bytes("verifySigs"), a.load() != Bytes("nop")), i.store(i.load() + Int(1))),

                # Now look through the whole group of 'verifySigs'
                While(i.load() <= Txn.group_index()).Do(Seq([
                            MagicAssert(And(
                                Gtxn[i.load()].type_enum() == TxnType.ApplicationCall,
                                Gtxn[i.load()].rekey_to() == Global.zero_address(),
                                Gtxn[i.load()].application_id() == Txn.application_id(),
                                Gtxn[i.load()].accounts[1] == Txn.accounts[1],
                                Gtxn[i.load()].accounts[2] == Txn.accounts[2],
                            )),
                            a.store(Gtxn[i.load()].application_args[0]),
                            Cond(
                                [a.load() == Bytes("verifySigs"), Seq([
                                    # Lets see if they are actually verifying the correct signatures!
                                    
                                    # What signatures did this verifySigs check?
                                    s.store(Gtxn[i.load()].application_args[1]),

                                    # Make sure we bail earlier on incorrect arguments...
                                    MagicAssert(Len(s.load()) > Int(0)),

                                    # Look at the vaa and confirm those were the expected signatures we should have been checking
                                    # at this point in the process
                                    MagicAssert(Extract(Txn.application_args[1], off.load(), Len(s.load())) == s.load()),

                                    # Where is the end pointer...
                                    eoff.store(off.load() + Len(s.load())),

                                    # Now we will reset s and collect the keys
                                    s.store(Bytes("")),

                                    While(off.load() < eoff.load()).Do(Seq( [
                                            # Lets see if we ever reuse the same signature more then once (same guardian over and over)
                                            guardian.store(Btoi(Extract(Txn.application_args[1], off.load(), Int(1)))),
                                            MagicAssert(GetBit(hits.load(), guardian.load()) == Int(0)),
                                            hits.store(SetBit(hits.load(), guardian.load(), Int(1))),

                                            # This extracts out of the keys THIS guardian's public key
                                            s.store(Concat(s.load(), Extract(guardian_keys.load(), guardian.load() * Int(20), Int(20)))),

                                            off.store(off.load() + Int(66))
                                    ])),

                                    MagicAssert(And(
                                        Gtxn[i.load()].application_args[2] == s.load(),      # Does the keyset passed into the verify routines match what it should be?
                                        Gtxn[i.load()].sender() == STATELESS_LOGIC_HASH,     # Was it signed with our code?
                                        Gtxn[i.load()].application_args[3] == digest.load()  # Was it verifying the same vaa?
                                    )),
                                    
                                ])],
                                [a.load() == Bytes("nop"), Seq([])],       # if there is a function call not listed here, it will throw an error
                                [a.load() == Bytes("verifyVAA"), Seq([])],
                                [Int(1) == Int(1), Seq([Reject()])]   # Nothing should get snuck in between...
                            ),
                            i.store(i.load() + Int(1))
                        ])
                ),

                # Did we verify all the signatures?  If the answer is no, something is sus
                MagicAssert(off.load() == Int(6) + (num_sigs.load() * Int(66))),

                Approve(),
            ])

        def governance():
            return Seq([
                checkForDuplicate(), # Verify this is not a duplicate message and then make sure we never see it again

                MagicAssert(And(
                    Gtxn[Txn.group_index() - Int(1)].type_enum() == TxnType.ApplicationCall,
                    Gtxn[Txn.group_index() - Int(1)].application_id() == Txn.application_id(),
                    Gtxn[Txn.group_index() - Int(1)].application_args[0] == Bytes("verifyVAA"),
                    Gtxn[Txn.group_index() - Int(1)].sender() == Txn.sender(),
                    Gtxn[Txn.group_index() - Int(1)].rekey_to() == Global.zero_address(),
                    Gtxn[Txn.group_index() - Int(1)].on_completion() == OnComplete.NoOp,

                    # Lets see if the vaa we are about to process was actually verified by the core
                    Gtxn[Txn.group_index() - Int(1)].application_args[1] == Txn.application_args[1],

                    # What checks should I give myself
                    Gtxn[Txn.group_index()].rekey_to() == Global.zero_address(),
                    Gtxn[Txn.group_index()].sender() == Txn.sender(),

                    # We all opted into the same accounts?
                    Gtxn[Txn.group_index() - Int(1)].accounts[0] == Txn.accounts[0],
                    Gtxn[Txn.group_index() - Int(1)].accounts[1] == Txn.accounts[1],
                    Gtxn[Txn.group_index() - Int(1)].accounts[2] == Txn.accounts[2],
                )),
                    
                hdlGovernance(Int(0)),
                Approve(),
            ])

        METHOD = Txn.application_args[0]

        on_delete = Seq([Reject()])

        router = Cond(
            [METHOD == Bytes("publishMessage"), publishMessage()],
            [METHOD == Bytes("nop"), nop()],
            [METHOD == Bytes("init"), init()],
            [METHOD == Bytes("verifySigs"), verifySigs()],
            [METHOD == Bytes("verifyVAA"), verifyVAA()],
            [METHOD == Bytes("governance"), governance()],
        )

        on_create = Seq( [
            App.globalPut(Bytes("MessageFee"), Int(0)),
            App.globalPut(Bytes("vphash"), Bytes("")),
            App.globalPut(Bytes("currentGuardianSetIndex"), Int(0)),
            App.globalPut(Bytes("validUpdateApproveHash"), Bytes("")),
            Return(Int(1))
        ])

        progHash = ScratchVar()
        progSet = ScratchVar()
        clearHash = ScratchVar()
        clearSet = ScratchVar()

        def getOnUpdate():
            return Seq( [
                MagicAssert(Sha512_256(Concat(Bytes("Program"), Txn.approval_program())) == App.globalGet(Bytes("validUpdateApproveHash"))),
                MagicAssert(And(Len(Txn.clear_state_program()) == Int(4), Extract(Txn.clear_state_program(), Int(1), Int(3)) == Bytes("base16", "810143"))),
                Return(Int(1))
            ] )

        on_update = getOnUpdate()
        
        on_optin = Seq( [
            Return(optin())
        ])

        return Cond(
            [Txn.application_id() == Int(0), on_create],
            [Txn.on_completion() == OnComplete.UpdateApplication, on_update],
            [Txn.on_completion() == OnComplete.DeleteApplication, on_delete],
            [Txn.on_completion() == OnComplete.OptIn, on_optin],
            [Txn.on_completion() == OnComplete.NoOp, router]
        )
    
    def clear_state_program():
        return Int(1)

    if not devMode:
        client = AlgodClient("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "https://testnet-api.algonode.cloud")
    APPROVAL_PROGRAM = fullyCompileContract(genTeal, client, vaa_processor_program(seed_amt, tmpl_sig), approve_name, devMode)
    CLEAR_STATE_PROGRAM = fullyCompileContract(genTeal, client, clear_state_program(), clear_name, devMode)

    return APPROVAL_PROGRAM, CLEAR_STATE_PROGRAM

def cli(output_approval, output_clear):
    seed_amt = 1002000
    tmpl_sig = TmplSig("sig")

    client = AlgodClient("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "https://testnet-api.algonode.cloud")

    approval, clear = getCoreContracts(True, output_approval, output_clear, client, seed_amt, tmpl_sig, True)

if __name__ == "__main__":
    cli(sys.argv[1], sys.argv[2])
