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

import pprint

max_keys = 16
max_bytes_per_key = 127
bits_per_byte = 8

bits_per_key = max_bytes_per_key * bits_per_byte
max_bytes = max_bytes_per_key * max_keys
max_bits = bits_per_byte * max_bytes

def fullyCompileContract(client: AlgodClient, contract: Expr) -> bytes:
    teal = compileTeal(contract, mode=Mode.Application, version=6)
    response = client.compile(teal)
    return response

def getCoreContracts(   client: AlgodClient,
                        seed_amt: int = 0,
                        tmpl_sig: TmplSig = None,
                        ) -> Tuple[bytes, bytes]:

    def vaa_processor_program(seed_amt: int, tmpl_sig: TmplSig):
        blob = LocalBlob()

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
                # EMMITTER_ID
                tmpl_sig.get_bytecode_chunk(1),
                encode_uvarint(Len(emitter), Bytes("")),
                emitter,
                # SEED_AMT
                tmpl_sig.get_bytecode_chunk(2),
                encode_uvarint(Int(seed_amt), Bytes("")),
                # APP_ID
                tmpl_sig.get_bytecode_chunk(3),
                encode_uvarint(Global.current_application_id(), Bytes("")),
                # TMPL_APP_ADDRESS
                tmpl_sig.get_bytecode_chunk(4),
                encode_uvarint(Len(Global.current_application_address()), Bytes("")),
                Global.current_application_address(),
                tmpl_sig.get_bytecode_chunk(5),
                )
            )
    
        @Subroutine(TealType.uint64)
        def optin():
            # Alias for readability
            algo_seed = Gtxn[0]
            optin = Gtxn[1]
    
            well_formed_optin = And(
                # Check that we're paying it
                algo_seed.type_enum() == TxnType.Payment,
                algo_seed.amount() == Int(seed_amt),
                # Check that its an opt in to us
                optin.type_enum() == TxnType.ApplicationCall,
                optin.on_completion() == OnComplete.OptIn,
                # Not strictly necessary since we wouldn't be seeing this unless it was us, but...
                optin.application_id() == Global.current_application_id(),
            )
    
            return Seq(
                # Make sure its a valid optin
                Assert(well_formed_optin),
                # Init by writing to the full space available for the sender (Int(0))
                blob.zero(Int(0)),
                # we gucci
                Int(1)
            )
    
        def nop():
            return Seq([Approve()])

        def publishMessage():
            seq = ScratchVar()

            return Seq([
                # Lets see if we were handed the correct account to store the sequence number in
                Assert(Txn.accounts[1] == get_sig_address(Int(0), Txn.sender())),

                # emitter sequence number
                seq.store(Itob(Btoi(blob.read(Int(1), Int(0), Int(8))) + Int(1))),
                Pop(blob.write(Int(1), Int(0), seq.load())),

                # Log it so that we can look for this on the guardian network
                Log(seq.load()),
                
                Approve()
#                Reject()
            ])

        def hdlGovernance():
            off = ScratchVar()
            a = ScratchVar()
            emitter = ScratchVar()
            idx = ScratchVar()
            set = ScratchVar()
            len = ScratchVar()
            v = ScratchVar()

            return Seq([
                # All governance must be done with the most recent guardian set
                set.store(App.globalGet(Bytes("currentGuardianSetIndex"))),
                If(set.load() != Bytes(""), Seq([
                        idx.store(Extract(Txn.application_args[1], Int(1), Int(4))),
                        Assert(idx.load() == set.load()),
                ])),

                # The offset of the chain
                off.store(Btoi(Extract(Txn.application_args[1], Int(5), Int(1))) * Int(66) + Int(14)), 
                # Correct chain? 
                Assert(Extract(Txn.application_args[1], off.load(), Int(2)) == Bytes("base16", "0001")),
                # Correct emitter?
                Assert(Extract(Txn.application_args[1], off.load() + Int(2), Int(32)) == Bytes("base16", "0000000000000000000000000000000000000000000000000000000000000004")),
                # Get us to the payload
                off.store(off.load() + Int(43)),
                # Is this a governance message?
                Assert(Extract(Txn.application_args[1], off.load(), Int(32)) == Bytes("base16", "00000000000000000000000000000000000000000000000000000000436f7265")),
                off.store(off.load() + Int(32)),
                a.store(Btoi(Extract(Txn.application_args[1], off.load(), Int(1)))),
                Cond( 
                    [a.load() == Int(1), Seq([
                        # ContractUpgrade is a VAA that instructs an implementation on a specific chain to upgrade itself
                        # 
                        # In the case of Algorand, it contains the hash of the program that we are allowed to upgrade ourselves to.  We would then run the upgrade program itself
                        # to perform the actual upgrade
                        Assert(Extract(Txn.application_args[1], off.load() + Int(1), Int(2)) == Bytes("base16", "0008")),
                        off.store(off.load() + Int(3)),

                        App.globalPut(Bytes("validUpdateApproveHash"), Extract(Txn.application_args[1], off.load(), Int(32)))
                    ])],
                    [a.load() == Int(2), Seq([
                        # We are updating the guardian set

                        # This should point at all chains
                        Assert(Extract(Txn.application_args[1], off.load() + Int(1), Int(2)) == Bytes("base16", "0000")),

                        # move off to point at the NewGuardianSetIndex and grab it
                        off.store(off.load() + Int(3)),
                        v.store(Extract(Txn.application_args[1], off.load(), Int(4))),
                        idx.store(Btoi(v.load())),

                        # Lets see if the user handed us the correct memory... no hacky hacky
                        Assert(Txn.accounts[3] == get_sig_address(idx.load(), Bytes("guardian"))), 

                        # Write this away till the next time
                        App.globalPut(Bytes("currentGuardianSetIndex"), v.load()),

                        # Write everything out to the auxilliary storage
                        off.store(off.load() + Int(4)),
                        len.store(Btoi(Extract(Txn.application_args[1], off.load(), Int(1)))),
                        Pop(blob.write(Int(3), Int(0), Extract(Txn.application_args[1], off.load(), Int(1) + (Int(20) * len.load()))))
                    ])]
                     ),
                Approve()
            ])

        def init():
            return Seq([
                # You better lose yourself in the music, the moment
                App.globalPut(Bytes("vphash"), Txn.application_args[2]),

                # You own it, you better never let it go
                Assert(Txn.sender() == Global.creator_address()),

                # You only get one shot, do not miss your chance to blow
                Assert(App.globalGet(Bytes("booted")) == Int(0)),
                App.globalPut(Bytes("booted"), Bytes("true")),

                # This opportunity comes once in a lifetime
                checkForDuplicate(),

                # You can do anything you set your mind to...
                hdlGovernance()
            ])

        def verifySigs():
            return Seq([
                Approve(),
            ])


        @Subroutine(TealType.none)
        def checkForDuplicate():
            off = ScratchVar()
            emitter = ScratchVar()
            sequence = ScratchVar()
            b = ScratchVar()
            byte_offset = ScratchVar()

            return Seq(
                # VM only is version 1
                Assert(Btoi(Extract(Txn.application_args[1], Int(0), Int(1))) == Int(1)),

                off.store(Btoi(Extract(Txn.application_args[1], Int(5), Int(1))) * Int(66) + Int(14)), # The offset of the emitter

                # emitter is chain/contract-address
                emitter.store(Extract(Txn.application_args[1], off.load(), Int(34))),
                sequence.store(Btoi(Extract(Txn.application_args[1], off.load() + Int(34), Int(8)))),

                # They passed us the correct account?  In this case, byte_offset points at the whole block
                byte_offset.store(sequence.load() / Int(max_bits)),
                Assert(Txn.accounts[1] == get_sig_address(byte_offset.load(), emitter.load())),

                # Now, lets go grab the raw byte
                byte_offset.store((sequence.load() / Int(8)) % Int(max_bytes)),
                b.store(blob.get_byte(Int(1), byte_offset.load())),

                # I would hope we've never seen this packet before...   throw an exception if we have
                Assert(GetBit(b.load(), sequence.load() % Int(8)) == Int(0)),

                # Lets mark this bit so that we never see it again
                blob.set_byte(Int(1), byte_offset.load(), SetBit(b.load(), sequence.load() % Int(8), Int(1)))
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
                Assert(Txn.accounts[2] == get_sig_address(Btoi(Extract(Txn.application_args[1], Int(1), Int(4))), Bytes("guardian"))),

                # Lets grab the total keyset
                total_guardians.store(blob.get_byte(Int(2), Int(0))),
                guardian_keys.store(blob.read(Int(2), Int(1), Int(1) + Int(20) * total_guardians.load())),
                hits.store(Bytes("base16", "0x00000000")),

                # How many signatures are in this vaa?
                num_sigs.store(Btoi(Extract(Txn.application_args[1], Int(5), Int(1)))),

                # Lets create a digest of THIS vaa...
                off.store(Int(6) + (num_sigs.load() * Int(66))),
                digest.store(Keccak256(Keccak256(Extract(Txn.application_args[1], off.load(), Len(Txn.application_args[1]) - off.load())))),

                # We have enough signatures?
                Assert(And(
                    total_guardians.load() > Int(0),
                    num_sigs.load() <= total_guardians.load(),
                    num_sigs.load() > ((total_guardians.load() * Int(2)) / Int(3)),
                    )),


                # There should always be 1 payment txid at the start for at least 3000 to the vphash...
                Assert(And(
                    Gtxn[0].type_enum() == TxnType.Payment,
                    Gtxn[0].amount() >= Int(3000),
                    Gtxn[0].receiver() == STATELESS_LOGIC_HASH
                )),

                # Point it at the start of the signatures in the VAA
                off.store(Int(6)),

                For(
                        i.store(Int(1)),
                        i.load() <= Txn.group_index(),
                        i.store(i.load() + Int(1))).Do(Seq([
                            Assert(And(
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
                                    s.store(Gtxn[i.load()].application_args[1]), # find a different way to get length
                                    Assert(Extract(Txn.application_args[1], off.load(), Len(s.load())) == s.load()),
                                    eoff.store(off.load() + Len(s.load())),

                                    s.store(Bytes("")),

                                    While(off.load() < eoff.load()).Do(Seq( [
                                            # Lets see if we ever reuse the same signature more then once (same guardian over and over)
                                            guardian.store(Btoi(Extract(Txn.application_args[1], off.load(), Int(1)))),
                                            Assert(GetBit(hits.load(), guardian.load()) == Int(0)),
                                            hits.store(SetBit(hits.load(), guardian.load(), Int(1))),
                                            
                                            s.store(Concat(s.load(), Extract(guardian_keys.load(), guardian.load() * Int(20), Int(20)))),

                                            off.store(off.load() + Int(66))
                                    ])),

                                    Assert(And(
                                        Gtxn[i.load()].application_args[2] == s.load(),      # Does the keyset passed into the verify routines match what it should be?
                                        Gtxn[i.load()].sender() == STATELESS_LOGIC_HASH,     # Was it signed with our code?
                                        Gtxn[i.load()].application_args[3] == digest.load()  # Was it verifying the same vaa?
                                    )),
                                    
                                ])],
                                [a.load() == Bytes("nop"), Seq([])],       # if there is a function call not listed here, it will throw an error
                                [a.load() == Bytes("verifyVAA"), Seq([])],
                            )
                        ])
                ),

                # Did we verify all the signatures?
                Assert(off.load() == Int(6) + (num_sigs.load() * Int(66))),

                Approve(),
            ])

        def governance():
            return Seq([
                checkForDuplicate(), # Verify this is not a duplicate message and then make sure we never see it again

                Assert(And(
                    Gtxn[Txn.group_index() - Int(1)].type_enum() == TxnType.ApplicationCall,
                    Gtxn[Txn.group_index() - Int(1)].application_id() == Txn.application_id(),
                    Gtxn[Txn.group_index() - Int(1)].application_args[0] == Bytes("verifyVAA"),
                    Gtxn[Txn.group_index() - Int(1)].sender() == Txn.sender(),
                    Gtxn[Txn.group_index() - Int(1)].rekey_to() == Global.zero_address(),

                    # Lets see if the vaa we are about to process was actually verified by the core
                    Gtxn[Txn.group_index() - Int(1)].application_args[1] == Txn.application_args[1],

                    # What checks should I give myself
                    Gtxn[Txn.group_index()].rekey_to() == Global.zero_address(),
                    Gtxn[Txn.group_index()].sender() == Txn.sender(),

                    # We all opted into the same accounts?
                    Gtxn[Txn.group_index() - Int(1)].accounts[0] == Txn.accounts[0],
                    Gtxn[Txn.group_index() - Int(1)].accounts[1] == Txn.accounts[1],
                    Gtxn[Txn.group_index() - Int(1)].accounts[2] == Txn.accounts[2],

                    (Global.group_size() - Int(1)) == Txn.group_index()    # governance should be the last entry...
                )),
                    
                hdlGovernance(),
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
            App.globalPut(Bytes("vphash"), Bytes("")),
            App.globalPut(Bytes("currentGuardianSetIndex"), Bytes("")),
            App.globalPut(Bytes("validUpdateApproveHash"), Bytes("")),
            App.globalPut(Bytes("validUpdateClearHash"), Bytes("BJATCHES5YJZJ7JITYMVLSSIQAVAWBQRVGPQUDT5AZ2QSLDSXWWM46THOY")), # empty clear state program
            Return(Int(1))
        ])

        on_update = Seq( [
            Assert(Sha512_256(Concat(Bytes("Program"), Txn.approval_program())) == App.globalGet(Bytes("validUpdateApproveHash"))),
            Assert(Sha512_256(Concat(Bytes("Program"), Txn.clear_state_program())) == App.globalGet(Bytes("validUpdateClearHash"))),
            Return(Int(1))
        ] )

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

    APPROVAL_PROGRAM = fullyCompileContract(client, vaa_processor_program(seed_amt, tmpl_sig))
    CLEAR_STATE_PROGRAM = fullyCompileContract(client, clear_state_program())

    return APPROVAL_PROGRAM, CLEAR_STATE_PROGRAM
