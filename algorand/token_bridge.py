#!/usr/bin/python3
"""
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

"""

from typing import List, Tuple, Dict, Any, Optional, Union

from pyteal.ast import *
from pyteal.types import *
from pyteal.compiler import *
from pyteal.ir import *
from globals import *
from inlineasm import *

from algosdk.v2client.algod import AlgodClient

from TmplSig import TmplSig

from local_blob import LocalBlob

import sys

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

def clear_token_bridge():
    return Int(1)

def approve_token_bridge(seed_amt: int, tmpl_sig: TmplSig):
    blob = LocalBlob()

    @Subroutine(TealType.bytes)
    def governanceSet() -> Expr:
        maybe = App.globalGetEx(App.globalGet(Bytes("coreid")), Bytes("currentGuardianSetIndex"))
        
        return Seq(maybe, Assert(maybe.hasValue()), maybe.value())
    
    
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
    def trim_bytes(str: Expr):
        len = ScratchVar()
        off = ScratchVar()
        zero = ScratchVar()
        r = ScratchVar()

        return Seq([
            r.store(str),

            len.store(Len(r.load())),
            zero.store(BytesZero(Int(1))),
            off.store(Int(0)),

            While(off.load() < len.load()).Do(Seq([
                If(Extract(r.load(), off.load(), Int(1)) == zero.load()).Then(Seq([
                        r.store(Extract(r.load(), Int(0), off.load())),
                        off.store(len.load())
                ])),
                    off.store(off.load() + Int(1))
            ])),
            r.load()
        ])

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

    def governance():
        off = ScratchVar()
        a = ScratchVar()
        targetChain = ScratchVar()
        chain = ScratchVar()
        emitter = ScratchVar()
        set = ScratchVar()
        idx = ScratchVar()

    
        return Seq([
            checkForDuplicate(),

            # All governance must be done with the most recent guardian set...
            set.store(governanceSet()),
            idx.store(Extract(Txn.application_args[1], Int(1), Int(4))),
            Assert(idx.load() == set.load()),

            # The offset of the chain
            off.store(Btoi(Extract(Txn.application_args[1], Int(5), Int(1))) * Int(66) + Int(14)), 

            Assert(And(
                # Did verifyVAA pass?
                Gtxn[Txn.group_index() - Int(1)].type_enum() == TxnType.ApplicationCall,
                Gtxn[Txn.group_index() - Int(1)].application_id() == App.globalGet(Bytes("coreid")),
                Gtxn[Txn.group_index() - Int(1)].application_args[0] == Bytes("verifyVAA"),
                Gtxn[Txn.group_index() - Int(1)].sender() == Txn.sender(),
                Gtxn[Txn.group_index() - Int(1)].rekey_to() == Global.zero_address(),

                # Lets see if the vaa we are about to process was actually verified by the core
                Gtxn[Txn.group_index() - Int(1)].application_args[1] == Txn.application_args[1],

                # What checks should I give myself
                Txn.rekey_to() == Global.zero_address(),

                # We all opted into the same accounts?
                Gtxn[Txn.group_index() - Int(1)].accounts[0] == Txn.accounts[0],
                Gtxn[Txn.group_index() - Int(1)].accounts[1] == Txn.accounts[1],
                Gtxn[Txn.group_index() - Int(1)].accounts[2] == Txn.accounts[2],

                # Better be the right emitters
                Extract(Txn.application_args[1], off.load(), Int(2)) == Bytes("base16", "0001"),
                Extract(Txn.application_args[1], off.load() + Int(2), Int(32)) == Bytes("base16", "0000000000000000000000000000000000000000000000000000000000000004"),
                
                (Global.group_size() - Int(1)) == Txn.group_index()    # This should be the last entry...
            )),

            off.store(off.load() + Int(43)),
            # correct module?
            Assert(Extract(Txn.application_args[1], off.load(), Int(32)) == Bytes("base16", "000000000000000000000000000000000000000000546f6b656e427269646765")),
            off.store(off.load() + Int(32)),
            a.store(Btoi(Extract(Txn.application_args[1], off.load(), Int(1)))),
            off.store(off.load() + Int(1)),

            Cond( 
                [a.load() == Int(1), Seq([
                    targetChain.store(Btoi(Extract(Txn.application_args[1], off.load(), Int(2)))),

                    Assert(Or((targetChain.load() == Int(0)), (targetChain.load() == Int(8)))),

                    off.store(off.load() + Int(2)),
                    chain.store(Extract(Txn.application_args[1], off.load(), Int(2))),

                    off.store(off.load() + Int(2)),
                    emitter.store(Extract(Txn.application_args[1], off.load(), Int(32))),

                    # Can I only register once?  Rumor says yes
                    Assert(App.globalGet(Concat(Bytes("Chain"), chain.load())) == Int(0)),

                    App.globalPut(Concat(Bytes("Chain"), chain.load()), emitter.load()),
                ])],
                [a.load() == Int(2), Seq([
                    Assert(Extract(Txn.application_args[1], off.load(), Int(2)) == Bytes("base16", "0008")),
                    off.store(off.load() + Int(2)),
                    App.globalPut(Bytes("validUpdateApproveHash"), Extract(Txn.application_args[1], off.load(), Int(32)))
                ])]
            ),

            Approve()
        ])
    
    def receiveAttest():
        me = Global.current_application_address()
        off = ScratchVar()
        
        Address = ScratchVar()
        Chain = ScratchVar()
        FromChain = ScratchVar()
        Decimals = ScratchVar()
        Symbol = ScratchVar()
        Name = ScratchVar()

        asset = ScratchVar()
        buf = ScratchVar()
        c = ScratchVar()
        a = ScratchVar()
        
        return Seq([
            checkForDuplicate(),

            Assert(And(
                # Lets see if the vaa we are about to process was actually verified by the core
                Gtxn[Txn.group_index() - Int(4)].type_enum() == TxnType.ApplicationCall,
                Gtxn[Txn.group_index() - Int(4)].application_id() == App.globalGet(Bytes("coreid")),
                Gtxn[Txn.group_index() - Int(4)].application_args[0] == Bytes("verifyVAA"),
                Gtxn[Txn.group_index() - Int(4)].sender() == Txn.sender(),
                Gtxn[Txn.group_index() - Int(4)].rekey_to() == Global.zero_address(),
                Gtxn[Txn.group_index() - Int(4)].application_args[1] == Txn.application_args[1],

                # We all opted into the same accounts?
                Gtxn[Txn.group_index() - Int(4)].accounts[0] == Txn.accounts[0],
                Gtxn[Txn.group_index() - Int(4)].accounts[1] == Txn.accounts[1],
                Gtxn[Txn.group_index() - Int(4)].accounts[2] == Txn.accounts[2],
                
                # Did the user pay the lsig to attest a new product?
                Gtxn[Txn.group_index() - Int(3)].type_enum() == TxnType.Payment,
                Gtxn[Txn.group_index() - Int(3)].amount() >= Int(100000),
                Gtxn[Txn.group_index() - Int(3)].sender() == Txn.sender(),
                Gtxn[Txn.group_index() - Int(3)].receiver() == Txn.accounts[3],
                Gtxn[Txn.group_index() - Int(3)].rekey_to() == Global.zero_address(),

                # We had to buy some extra CPU
                Gtxn[Txn.group_index() - Int(2)].type_enum() == TxnType.ApplicationCall,
                Gtxn[Txn.group_index() - Int(2)].application_id() == Global.current_application_id(),
                Gtxn[Txn.group_index() - Int(2)].application_args[0] == Bytes("nop"),
                Gtxn[Txn.group_index() - Int(2)].sender() == Txn.sender(),
                Gtxn[Txn.group_index() - Int(2)].rekey_to() == Global.zero_address(),

                Gtxn[Txn.group_index() - Int(1)].type_enum() == TxnType.ApplicationCall,
                Gtxn[Txn.group_index() - Int(1)].application_id() == Global.current_application_id(),
                Gtxn[Txn.group_index() - Int(1)].application_args[0] == Bytes("nop"),
                Gtxn[Txn.group_index() - Int(1)].sender() == Txn.sender(),
                Gtxn[Txn.group_index() - Int(1)].rekey_to() == Global.zero_address(),
                
                (Global.group_size() - Int(1)) == Txn.group_index()    # This should be the last entry...
            )),

            off.store(Btoi(Extract(Txn.application_args[1], Int(5), Int(1))) * Int(66) + Int(6) + Int(8)), # The offset of the chain
            Chain.store(Btoi(Extract(Txn.application_args[1], off.load(), Int(2)))),

            # Make sure that the emitter on the sending chain is correct for the token bridge
            Assert(App.globalGet(Concat(Bytes("Chain"), Extract(Txn.application_args[1], off.load(), Int(2)))) 
                   == Extract(Txn.application_args[1], off.load() + Int(2), Int(32))),
            
            off.store(off.load()+Int(43)),

            Assert(Int(2) ==      Btoi(Extract(Txn.application_args[1], off.load(),           Int(1)))),
            Address.store(             Extract(Txn.application_args[1], off.load() + Int(1),  Int(32))),
            
            FromChain.store(      Btoi(Extract(Txn.application_args[1], off.load() + Int(33), Int(2)))),
            Decimals.store(       Btoi(Extract(Txn.application_args[1], off.load() + Int(35), Int(1)))),
            Symbol.store(              Extract(Txn.application_args[1], off.load() + Int(36), Int(32))),
            Name.store(                Extract(Txn.application_args[1], off.load() + Int(68), Int(32))),

            # Lets trim this... seems these are limited to 7 characters
            Symbol.store(trim_bytes(Symbol.load())),
            If (Len(Symbol.load()) > Int(7), Symbol.store(Extract(Symbol.load(), Int(0), Int(7)))),
            Name.store(trim_bytes(Name.load())),

            # Due to constrains on some supported chains, all token
            # amounts passed through the token bridge are truncated to
            # a maximum of 8 decimals. 
            # 
            # Any chains implementation must make sure that of any
            # token only ever MaxUint64 units (post-shifting) are
            # bridged into the wormhole network at any given time (all
            # target chains combined), even tough the slot is 32 bytes
            # long (theoretically fitting uint256). 
            If(Decimals.load() > Int(8), Decimals.store(Int(8))),

            #   This confirms the user gave us access to the correct memory for this asset..
            Assert(Txn.accounts[3] == get_sig_address(FromChain.load(), Address.load())),

            # Lets see if we've seen this asset before
            asset.store(blob.read(Int(3), Int(0), Int(8))),

            # The # offset to the digest
            off.store(Btoi(Extract(Txn.application_args[1], Int(5), Int(1))) * Int(66) + Int(6)), 

            # New asset
            If(asset.load() == Itob(Int(0))).Then(Seq([
                    InnerTxnBuilder.Begin(),
                    InnerTxnBuilder.SetFields(
                        {
                            TxnField.sender: Txn.accounts[3],
                            TxnField.type_enum: TxnType.AssetConfig,
                            TxnField.config_asset_name: Name.load(),
                            TxnField.config_asset_unit_name: Symbol.load(),
                            TxnField.config_asset_total: Int(int(1e17)),
                            TxnField.config_asset_decimals: Decimals.load(),
                            TxnField.config_asset_manager: me,
                            TxnField.config_asset_reserve: me,

                        # We cannot freeze or clawback assets... per the spirit of 
                        TxnField.config_asset_freeze: Global.zero_address(),
                            TxnField.config_asset_clawback: Global.zero_address(),

                        TxnField.fee: Int(0),
                        }
                    ),
                    InnerTxnBuilder.Submit(),

                asset.store(Itob(InnerTxn.created_asset_id())),
                    Pop(blob.write(Int(3), Int(0), asset.load())),
            ])).Else(Seq([
                Assert(And(
                    # Same address?
                    blob.read(Int(3), Int(60), Int(92)) == Address.load(),
                    # Same chain
                    Btoi(blob.read(Int(3), Int(92), Int(94))) == Chain.load()
                )),

                # I've been told we are supposted to update the name if we see it a second time
                Name.store(trim_bytes(Name.load())),
                Symbol.store(trim_bytes(Symbol.load())),

                InnerTxnBuilder.Begin(),
                InnerTxnBuilder.SetFields(
                    {
                        TxnField.type_enum: TxnType.AssetConfig,
                        TxnField.config_asset: Btoi(asset.load()),
                        TxnField.config_asset_name: Name.load(),
                        TxnField.config_asset_unit_name: Symbol.load(),
                        TxnField.fee: Int(0)
                    }
                ),
                InnerTxnBuilder.Submit(),
            ])),
            
            # We save away the entire digest that created this asset in case we ever need to reproduce it while sending this
            # coin to another chain

            buf.store(Txn.application_args[1]),
            Pop(blob.write(Int(3), Int(8), Extract(buf.load(), off.load(), Len(buf.load()) - off.load()))),

            Approve()
        ])

    def receiveTransfer():
        me = Global.current_application_address()
        off = ScratchVar()
        
        Chain = ScratchVar()
        Emitter = ScratchVar()

        Amount = ScratchVar()
        Origin = ScratchVar()
        OriginChain = ScratchVar()
        Destination = ScratchVar()
        DestChain = ScratchVar()
        Fee = ScratchVar()
        asset = ScratchVar()

        factor = ScratchVar()
        d = ScratchVar()
        zb = ScratchVar()
        action = ScratchVar()
        
        return Seq([
            checkForDuplicate(),

            zb.store(Bytes("base16", "0000000000000000000000000000000000000000000000000000000000000000")),


            Assert(And(
                # Lets see if the vaa we are about to process was actually verified by the core
                Gtxn[Txn.group_index() - Int(1)].type_enum() == TxnType.ApplicationCall,
                Gtxn[Txn.group_index() - Int(1)].application_id() == App.globalGet(Bytes("coreid")),
                Gtxn[Txn.group_index() - Int(1)].application_args[0] == Bytes("verifyVAA"),
                Gtxn[Txn.group_index() - Int(1)].sender() == Txn.sender(),
                Gtxn[Txn.group_index() - Int(1)].rekey_to() == Global.zero_address(),
                Gtxn[Txn.group_index() - Int(1)].application_args[1] == Txn.application_args[1],

                # We all opted into the same accounts?
                Gtxn[Txn.group_index() - Int(1)].accounts[0] == Txn.accounts[0],
                Gtxn[Txn.group_index() - Int(1)].accounts[1] == Txn.accounts[1],
                Gtxn[Txn.group_index() - Int(1)].accounts[2] == Txn.accounts[2],
                
                (Global.group_size() - Int(1)) == Txn.group_index()    # This should be the last entry...
            )),

            off.store(Btoi(Extract(Txn.application_args[1], Int(5), Int(1))) * Int(66) + Int(6) + Int(8)), # The offset of the chain

            Chain.store(Btoi(Extract(Txn.application_args[1], off.load(), Int(2)))),
            Emitter.store(Extract(Txn.application_args[1], off.load() + Int(2), Int(32))),

            # We coming from the correct emitter on the sending chain for the token bridge
            # ... This is 90% of the security...
            If(Chain.load() == Int(8),
               Assert(Global.current_application_address() == Emitter.load()), # This came from us?
               Assert(App.globalGet(Concat(Bytes("Chain"), Extract(Txn.application_args[1], off.load(), Int(2)))) == Emitter.load())),

            off.store(off.load()+Int(43)),

            # This is a transfer message... right?
            action.store(Btoi(Extract(Txn.application_args[1], off.load(), Int(1)))),

            Assert(Or(action.load() == Int(1), action.load() == Int(3))),

            Assert(Extract(Txn.application_args[1], off.load() + Int(1), Int(24)) == Extract(zb.load(), Int(0), Int(24))),
            Amount.store(        Btoi(Extract(Txn.application_args[1], off.load() + Int(25), Int(8)))),  # uint256

            Origin.store(             Extract(Txn.application_args[1], off.load() + Int(33), Int(32))),
            OriginChain.store(   Btoi(Extract(Txn.application_args[1], off.load() + Int(65), Int(2)))),
            Destination.store(        Extract(Txn.application_args[1], off.load() + Int(67), Int(32))),
            DestChain.store(     Btoi(Extract(Txn.application_args[1], off.load() + Int(99), Int(2)))),

            Assert(Extract(Txn.application_args[1], off.load() + Int(101),Int(24)) == Extract(zb.load(), Int(0), Int(24))),
            Fee.store(           Btoi(Extract(Txn.application_args[1], off.load() + Int(125),Int(8)))),  # uint256

            # This directed at us?
            Assert(DestChain.load() == Int(8)),

            If (action.load() == Int(3), Assert(Destination.load() == Txn.sender())),

            If(OriginChain.load() == Int(8),
               Seq([
                   asset.store(Btoi(Extract(Origin.load(), Int(24), Int(8)))),
                   Assert(Txn.accounts[3] == get_sig_address(asset.load(), Bytes("native"))),
                   # Now, the horrible part... we have to scale the amount back out to compensate for the "dedusting" 
                   # when this was sent...

                   If(asset.load() == Int(0),
                      Seq([
                          InnerTxnBuilder.Begin(),
                          InnerTxnBuilder.SetFields(
                              {
                                  TxnField.sender: Txn.accounts[3],
                                  TxnField.type_enum: TxnType.Payment,
                                  TxnField.receiver: Destination.load(),
                                  TxnField.amount: Amount.load(),
                                  TxnField.fee: Int(0),
                              }
                          ),
                          InnerTxnBuilder.Submit(),

                          If(Fee.load() > Int(0), Seq([
                                  InnerTxnBuilder.Begin(),
                                  InnerTxnBuilder.SetFields(
                                      {
                                          TxnField.sender: Txn.accounts[3],
                                          TxnField.type_enum: TxnType.Payment,
                                          TxnField.receiver: Txn.sender(),
                                          TxnField.amount: Fee.load(),
                                          TxnField.fee: Int(0),
                                      }
                                  ),
                                  InnerTxnBuilder.Submit(),
                          ])),

                          Approve()
                      ]),            # End of special case for algo
                      Seq([          # Start of handling code for algorand tokens
                          factor.store(Int(1)),
                          d.store(Btoi(extract_decimal(asset.load()))),
                          Cond(
                              [d.load() == Int(9),  factor.store(Int(10))],
                              [d.load() == Int(10), factor.store(Int(100))],
                              [d.load() == Int(11), factor.store(Int(1000))],
                              [d.load() == Int(12), factor.store(Int(10000))],
                              [d.load() == Int(13), factor.store(Int(100000))],
                              [d.load() == Int(14), factor.store(Int(1000000))],
                              [d.load() == Int(15), factor.store(Int(10000000))],
                              [d.load() == Int(16), factor.store(Int(100000000))],
                              [d.load() >  Int(16), Assert(d.load() < Int(16))],
                          ),
                          If(factor.load() != Int(1),
                             Seq([
                                 Amount.store(Amount.load() * factor.load()),
                                 Fee.store(Fee.load() * factor.load())
                             ])
                          ),       # If(factor.load() != Int(1),
                      ])           # End of handling code for algorand tokens
                   ),              # If(asset.load() == Int(0),
               ]),                 # If(OriginChain.load() == Int(8),

               # OriginChain.load() != Int(8),
               Seq([
                   # Lets see if we've seen this asset before
                   asset.store(Btoi(blob.read(Int(3), Int(0), Int(8)))),
                   Assert(And(
                       asset.load() != Int(0),
                       Txn.accounts[3] == get_sig_address(OriginChain.load(), Origin.load())
                     )
                   ),
               ])  # OriginChain.load() != Int(8),
            ),  #  If(OriginChain.load() == Int(8)


            # Actually send the coins...
#            Log(Bytes("Main")),
            InnerTxnBuilder.Begin(),
            InnerTxnBuilder.SetFields(
                {
                    TxnField.sender: Txn.accounts[3],
                    TxnField.type_enum: TxnType.AssetTransfer,
                    TxnField.xfer_asset: asset.load(),
                    TxnField.asset_amount: Amount.load(),
                    TxnField.asset_receiver: Destination.load(),
                    TxnField.fee: Int(0),
                }
            ),
            InnerTxnBuilder.Submit(),

            If(Fee.load() > Int(0), Seq([
#                    Log(Bytes("Fees")),
                    InnerTxnBuilder.Begin(),
                    InnerTxnBuilder.SetFields(
                        {
                            TxnField.sender: Txn.accounts[3],
                            TxnField.type_enum: TxnType.AssetTransfer,
                            TxnField.xfer_asset: asset.load(),
                            TxnField.asset_amount: Fee.load(),
                            TxnField.asset_receiver: Txn.sender(),
                            TxnField.fee: Int(0),
                        }
                    ),
                    InnerTxnBuilder.Submit(),
            ])),

            Approve()
        ])

    METHOD = Txn.application_args[0]

    on_delete = Seq([Reject()])

    @Subroutine(TealType.bytes)
    def auth_addr(id) -> Expr:
        maybe = AccountParam.authAddr(id)
        return Seq(maybe, If(maybe.hasValue(), maybe.value(), Bytes("")))

    @Subroutine(TealType.bytes)
    def extract_name(id) -> Expr:
        maybe = AssetParam.name(id)
        return Seq(maybe, If(maybe.hasValue(), maybe.value(), Bytes("")))

    @Subroutine(TealType.bytes)
    def extract_creator(id) -> Expr:
        maybe = AssetParam.creator(id)
        return Seq(maybe, If(maybe.hasValue(), maybe.value(), Bytes("")))

    @Subroutine(TealType.bytes)
    def extract_unit_name(id) -> Expr:
        maybe = AssetParam.unitName(id)
        return Seq(maybe, If(maybe.hasValue(), maybe.value(), Bytes("")))

    @Subroutine(TealType.bytes)
    def extract_decimal(id) -> Expr:
        maybe = AssetParam.decimals(id)
        return Seq(maybe, If(maybe.hasValue(), Extract(Itob(maybe.value()), Int(7), Int(1)), Bytes("base16", "00")))


    def sendTransfer():
        aid = ScratchVar()
        amount = ScratchVar()
        d = ScratchVar()
        p = ScratchVar()
        asset = ScratchVar()
        aaddr = ScratchVar()
        Address = ScratchVar()
        FromChain = ScratchVar()
        zb = ScratchVar()
        factor = ScratchVar()
        fee = ScratchVar()

        return Seq([
            zb.store(Bytes("base16", "0000000000000000000000000000000000000000000000000000000000000000")),

            aid.store(Btoi(Txn.application_args[1])),
            Assert(And(
                # We dont know what we don't know.   Is it readonable
                # to be so restrictive in a wormhole asset transfer?
                # to not let you put more crap in the same txn block?
                Txn.group_index() == Int(1),
                Global.group_size() == Int(2),
                Len(Txn.application_args[3]) <= Int(32)
            )),

            aid.store(Btoi(Txn.application_args[1])),

            # what should we pass as a fee...
            fee.store(Btoi(Txn.application_args[5])),

            If(aid.load() == Int(0),
               Seq([
                   Assert(And(
                       # The previous txn is the asset transfer itself
                       Gtxn[Txn.group_index() - Int(1)].type_enum() == TxnType.Payment,
                       Gtxn[Txn.group_index() - Int(1)].sender() == Txn.sender(),
                       Gtxn[Txn.group_index() - Int(1)].receiver() == Txn.accounts[2],
                       Gtxn[Txn.group_index() - Int(1)].rekey_to() == Global.zero_address(),
                   )),
                   amount.store(Gtxn[Txn.group_index() - Int(1)].amount()),
                   Assert(fee.load() < amount.load()),
                   amount.store(amount.load() - fee.load())
               ]),
               Seq([
                   Assert(And(
                       # The previous txn is the asset transfer itself
                       Gtxn[Txn.group_index() - Int(1)].type_enum() == TxnType.AssetTransfer,
                       Gtxn[Txn.group_index() - Int(1)].sender() == Txn.sender(),
                       Gtxn[Txn.group_index() - Int(1)].xfer_asset() == aid.load(),
                       Gtxn[Txn.group_index() - Int(1)].asset_receiver() == Txn.accounts[2],
                       Gtxn[Txn.group_index() - Int(1)].rekey_to() == Global.zero_address(),
                   )),
                   amount.store(Gtxn[Txn.group_index() - Int(1)].asset_amount()),

                   # peal the fee off the amount
                   Assert(fee.load() < amount.load()),
                   amount.store(amount.load() - fee.load()),

                   d.store(Btoi(extract_decimal(aid.load()))),

                   Assert(d.load() >= Int(0)),

                   factor.store(Int(1)),
                   Cond(
                       [d.load() == Int(9),  factor.store(Int(10))],
                       [d.load() == Int(10), factor.store(Int(100))],
                       [d.load() == Int(11), factor.store(Int(1000))],
                       [d.load() == Int(12), factor.store(Int(10000))],
                       [d.load() == Int(13), factor.store(Int(100000))],
                       [d.load() == Int(14), factor.store(Int(1000000))],
                       [d.load() == Int(15), factor.store(Int(10000000))],
                       [d.load() == Int(16), factor.store(Int(100000000))],
                       [d.load() >  Int(16), Assert(d.load() < Int(16))],
                   ),

                   If(factor.load() != Int(1),
                      Seq([
                          amount.store(amount.load() / factor.load()),
                          fee.store(fee.load() / factor.load()),
                      ])
                    ),       # If(factor.load() != Int(1),
               ]),
            ),

            # If it is nothing but dust lets just abort the whole transaction and save 
            Assert(And(amount.load() > Int(0), fee.load() >= Int(0))),

            If(aid.load() != Int(0),
               aaddr.store(auth_addr(extract_creator(aid.load()))),
               aaddr.store(Bytes(""))),
            
            # Is the authorizing signature of the creator of the asset the address of the token_bridge app itself?
            If(aaddr.load() == Global.current_application_address(),
               Seq([
#                   Log(Bytes("Wormhole wrapped")),

                   asset.store(blob.read(Int(2), Int(0), Int(8))),
                   # This the correct asset?
                   Assert(Txn.application_args[1] == asset.load()),

                   # Pull the address and chain out of the original vaa
                   Address.store(blob.read(Int(2), Int(60), Int(92))),
                   FromChain.store(blob.read(Int(2), Int(92), Int(94))),

                   # This the correct page given the chain and the address
                   Assert(Txn.accounts[2] == get_sig_address(Btoi(FromChain.load()), Address.load())),
               ]),
               Seq([
#                   Log(Bytes("Non Wormhole wrapped")),
                   Assert(Txn.accounts[2] == get_sig_address(aid.load(), Bytes("native"))),
                   FromChain.store(Bytes("base16", "0008")),
                   Address.store(Txn.application_args[1]),
               ])
            ),

            # Correct address len?
            Assert(And(
                Len(Address.load()) <= Int(32),
                Len(FromChain.load()) == Int(2),
                Len(Txn.application_args[3]) <= Int(32)
            )),

            p.store(Concat(
                If(Txn.application_args.length() == Int(6),
                   Bytes("base16", "01"),
                   Bytes("base16", "03")),
                Extract(zb.load(), Int(0), Int(24)),
                Itob(amount.load()),  # 8 bytes
                Extract(zb.load(), Int(0), Int(32) - Len(Address.load())),
                Address.load(),
                FromChain.load(),
                Extract(zb.load(), Int(0), Int(32) - Len(Txn.application_args[3])),
                Txn.application_args[3],
                Extract(Txn.application_args[4], Int(6), Int(2)),
                Extract(zb.load(), Int(0), Int(24)),
                Itob(fee.load()),  # 8 bytes
                If(Txn.application_args.length() == Int(7), Txn.application_args[6], Bytes(""))
            )),

            # This one magic line should protect us from overruns/underruns and trickery..
            If(Txn.application_args.length() == Int(7), 
               Assert(Len(p.load()) == Int(133) + Len(Txn.application_args[6])),
               Assert(Len(p.load()) == Int(133))),

            InnerTxnBuilder.Begin(),
            InnerTxnBuilder.SetFields(
                {
                    TxnField.type_enum: TxnType.ApplicationCall,
                    TxnField.application_id: App.globalGet(Bytes("coreid")),
                    TxnField.application_args: [Bytes("publishMessage"), p.load()],
                    TxnField.accounts: [Txn.accounts[1]],
                    TxnField.note: Bytes("publishMessage"),
                    TxnField.fee: Int(0),
                }
            ),
            InnerTxnBuilder.Submit(),

            Approve()
        ])

    def do_optin():
        return Seq([
            Assert(And(
                Txn.rekey_to() == Global.zero_address(),
                Txn.accounts[1] == get_sig_address(Btoi(Txn.application_args[1]), Bytes("native"))
            )),

            InnerTxnBuilder.Begin(),
            InnerTxnBuilder.SetFields(
                {
                    TxnField.sender: Txn.accounts[1],
                    TxnField.type_enum: TxnType.AssetTransfer,
                    TxnField.xfer_asset: Btoi(Txn.application_args[1]),
                    TxnField.asset_amount: Int(0),
                    TxnField.asset_receiver: Txn.accounts[1],
                    TxnField.fee: Int(0),
                }
            ),
            InnerTxnBuilder.Submit(),

            Approve()
        ])

    def transferWithPayload():
        return Seq([Approve()])

    # This is for attesting
    def attestToken():
        asset = ScratchVar()
        p = ScratchVar()
        zb = ScratchVar()
        d = ScratchVar()
        uname = ScratchVar()
        name = ScratchVar()
        aid = ScratchVar()

        Address = ScratchVar()
        FromChain = ScratchVar()

        return Seq([
            aid.store(Btoi(Txn.application_args[1])),
            # Is the authorizing signature of the creator of the asset the address of the token_bridge app itself?
            If(auth_addr(extract_creator(aid.load())) == Global.current_application_address(),
               Seq([
#                   Log(Bytes("Wormhole wrapped")),
                   # Wormhole wrapped asset
                   asset.store(blob.read(Int(2), Int(0), Int(8))),
                   # This the correct asset?
                   Assert(Txn.application_args[1] == asset.load()),

                   # Pull the address and chain out of the original vaa
                   Address.store(blob.read(Int(2), Int(60), Int(92))),
                   FromChain.store(Btoi(blob.read(Int(2), Int(92), Int(94)))),

                   # This the correct page given the chain and the address
                   Assert(Txn.accounts[2] == get_sig_address(FromChain.load(), Address.load())),

                   # this is wormhole wrapped... it shouldn't be busting 8 
                   Assert(Btoi(extract_decimal(aid.load())) <= Int(8)),

                   # Lets just hand back the previously generated vaa payload
                   p.store(blob.read(Int(2), Int(8), Int(108)))
               ]),
               Seq([
#                   Log(Bytes("Non Wormhole wrapped")),
                   Assert(Txn.accounts[2] == get_sig_address(aid.load(), Bytes("native"))),

                   zb.store(Bytes("base16", "0000000000000000000000000000000000000000000000000000000000000000")),
                   
                   aid.store(Btoi(Txn.application_args[1])),

                   If(aid.load() == Int(0),
                      Seq([
                          d.store(Bytes("base16", "06")),
                          uname.store(Bytes("ALGO")),
                          name.store(Bytes("ALGO"))
                      ]),
                      Seq([
                          d.store(extract_decimal(aid.load())),
                          If(Btoi(d.load()) > Int(8), d.store(Bytes("base16", "08"))),
                          uname.store(extract_unit_name(aid.load())),
                          name.store(extract_name(aid.load())),
                      ])
                    ),

                   p.store(
                       Concat(
                           #PayloadID uint8 = 2
                           Bytes("base16", "02"),
                           #TokenAddress [32]uint8
                           Extract(zb.load(),Int(0), Int(24)),
                           Txn.application_args[1],
                           #TokenChain uint16
                           Bytes("base16", "0008"),
                           #Decimals uint8
                           d.load(),
                           #Symbol [32]uint8
                           uname.load(),
                           Extract(zb.load(), Int(0), Int(32) - Len(uname.load())),
                           #Name [32]uint8
                           name.load(),
                           Extract(zb.load(), Int(0), Int(32) - Len(name.load())),
                       )
                   ),
               ])
               ),

            Assert(Len(p.load()) == Int(100)),

            InnerTxnBuilder.Begin(),
            InnerTxnBuilder.SetFields(
                {
                    TxnField.type_enum: TxnType.ApplicationCall,
                    TxnField.application_id: App.globalGet(Bytes("coreid")),
                    TxnField.application_args: [Bytes("publishMessage"), p.load()],
                    TxnField.accounts: [Txn.accounts[1]],
                    TxnField.note: Bytes("publishMessage"),
                    TxnField.fee: Int(0),
                }
            ),
            InnerTxnBuilder.Submit(),

            Approve()
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

    def nop():
        return Seq([Approve()])

    router = Cond(
        [METHOD == Bytes("nop"), nop()],
        [METHOD == Bytes("receiveAttest"), receiveAttest()],
        [METHOD == Bytes("attestToken"), attestToken()],
        [METHOD == Bytes("receiveTransfer"), receiveTransfer()],
        [METHOD == Bytes("sendTransfer"), sendTransfer()],
        [METHOD == Bytes("optin"), do_optin()],
        [METHOD == Bytes("transferWithPayload"), transferWithPayload()],
        [METHOD == Bytes("governance"), governance()],
    )

    on_create = Seq( [
        App.globalPut(Bytes("coreid"), Btoi(Txn.application_args[0])),
        App.globalPut(Bytes("validUpdateApproveHash"), Bytes("")),
        App.globalPut(Bytes("validUpdateClearHash"), Bytes("BJATCHES5YJZJ7JITYMVLSSIQAVAWBQRVGPQUDT5AZ2QSLDSXWWM46THOY")), # empty clear state program
        Return(Int(1))
    ])

    on_update = Seq( [
        Assert(Sha512_256(Concat(Bytes("Program"), Txn.approval_program())) == App.globalGet(Bytes("validUpdateApproveHash"))),
        Assert(Sha512_256(Concat(Bytes("Program"), Txn.clear_state_program())) == App.globalGet(Bytes("validUpdateClearHash"))),
        Return(Int(1))
    ] )

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

def get_token_bridge(client: AlgodClient, seed_amt: int = 0, tmpl_sig: TmplSig = None) -> Tuple[bytes, bytes]:
    APPROVAL_PROGRAM = fullyCompileContract(client, approve_token_bridge(seed_amt, tmpl_sig))
    CLEAR_STATE_PROGRAM = fullyCompileContract(client, clear_token_bridge())

    return APPROVAL_PROGRAM, CLEAR_STATE_PROGRAM
