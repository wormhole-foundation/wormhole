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
import pprint

from local_blob import LocalBlob

from algosdk.v2client.algod import AlgodClient
from algosdk.kmd import KMDClient
from algosdk import account, mnemonic
from algosdk.encoding import decode_address
from algosdk.future import transaction
from pyteal import compileTeal, Mode, Expr
from pyteal import *
from algosdk.logic import get_application_address

from algosdk.future.transaction import LogicSigAccount

class TmplSig:
    """KeySig class reads in a json map containing assembly details of a template smart signature and allows you to populate it with the variables
    In this case we are only interested in a single variable, the key which is a byte string to make the address unique.
    In this demo we're using random strings but in practice you can choose something meaningful to your application
    """

    def __init__(self, name):
        # Read the source map

#        with open("{}.json".format(name)) as f:
#            self.map = json.loads(f.read())

        self.map = {
            'bytecode': 'BiABAYEASIAASIgAAUMyBIEDEjMAECISEDMACIEAEhAzACAyAxIQMwAJMgMSEDMBEIEGEhAzARkiEhAzARiBABIQMwEgMgMSEDMCECISEDMCCIEAEhAzAiCAABIQMwIJMgMSEIk=',
            'label_map': {'init_0': 9},
            'name': 'sig.teal',
            'template_labels': {'TMPL_ADDR_IDX': {'bytes': False,
                                                  'position': 5,
                                                  'source_line': 3},
                                'TMPL_APP_ADDRESS': {'bytes': True,
                                                     'position': 90,
                                                     'source_line': 55},
                                'TMPL_APP_ID': {'bytes': False,
                                                'position': 63,
                                                'source_line': 39},
                                'TMPL_EMITTER_ID': {'bytes': True,
                                                    'position': 8,
                                                    'source_line': 5},
                                'TMPL_SEED_AMT': {'bytes': False,
                                                  'position': 29,
                                                  'source_line': 19}},
            'version': 6}

        self.src = base64.b64decode(self.map["bytecode"])
        self.sorted = dict(
            sorted(
                self.map["template_labels"].items(),
                key=lambda item: item[1]["position"],
            )
        )

    def populate(self, values: Dict[str, Union[str, int]]) -> LogicSigAccount:
        """populate uses the map to fill in the variable of the bytecode and returns a logic sig with the populated bytecode"""
        # Get the template source
        contract = list(base64.b64decode(self.map["bytecode"]))

        shift = 0
        for k, v in self.sorted.items():
            if k in values:
                pos = v["position"] + shift
                if v["bytes"]:
                    val = bytes.fromhex(values[k])
                    lbyte = uvarint.encode(len(val))
                    # -1 to account for the existing 00 byte for length
                    shift += (len(lbyte) - 1) + len(val)
                    # +1 to overwrite the existing 00 byte for length
                    contract[pos : pos + 1] = lbyte + val
                else:
                    val = uvarint.encode(values[k])
                    # -1 to account for existing 00 byte
                    shift += len(val) - 1
                    # +1 to overwrite existing 00 byte
                    contract[pos : pos + 1] = val

        # Create a new LogicSigAccount given the populated bytecode,

#        pprint.pprint({"values": values, "contract": bytes(contract).hex()})

        return LogicSigAccount(bytes(contract))

    def get_bytecode_chunk(self, idx: int) -> Bytes:
        start = 0
        if idx > 0:
            start = list(self.sorted.values())[idx - 1]["position"] + 1

        stop = len(self.src)
        if idx < len(self.sorted):
            stop = list(self.sorted.values())[idx]["position"]

        chunk = self.src[start:stop]
        return Bytes(chunk)

    def get_sig_tmpl(self):
        def sig_tmpl():
            # We encode the app id as an 8 byte integer to ensure its a known size
            # Otherwise the uvarint encoding may produce a different byte offset
            # for the template variables
            admin_app_id = Tmpl.Int("TMPL_APP_ID")
            admin_address = Tmpl.Bytes("TMPL_APP_ADDRESS")
            seed_amt = Tmpl.Int("TMPL_SEED_AMT")
        
            @Subroutine(TealType.uint64)
            def init():
                algo_seed = Gtxn[0]
                optin = Gtxn[1]
                rekey = Gtxn[2]

                return Seq([
                    Assert(Global.group_size() == Int(3)),
        
                    Assert(algo_seed.type_enum() == TxnType.Payment),
                    Assert(algo_seed.amount() == seed_amt),
                    Assert(algo_seed.rekey_to() == Global.zero_address()),
                    Assert(algo_seed.close_remainder_to() == Global.zero_address()),
        
                    Assert(optin.type_enum() == TxnType.ApplicationCall),
                    Assert(optin.on_completion() == OnComplete.OptIn),
                    Assert(optin.application_id() == admin_app_id),
                    Assert(optin.rekey_to() == Global.zero_address()),
        
                    Assert(rekey.type_enum() == TxnType.Payment),
                    Assert(rekey.amount() == Int(0)),
                    Assert(rekey.rekey_to() == admin_address),
                    Assert(rekey.close_remainder_to() == Global.zero_address()),
                    Approve()
                ])
        
            return Seq(
                # Just putting adding this as a tmpl var to make the address unique and deterministic
                # We don't actually care what the value is, pop it
                Pop(Tmpl.Int("TMPL_ADDR_IDX")),
                Pop(Tmpl.Bytes("TMPL_EMITTER_ID")),
                init(),
            )
        
        return compileTeal(sig_tmpl(), mode=Mode.Signature, version=6, assembleConstants=True)

if __name__ == '__main__':
    core = TmplSig("sig")
#    client =  AlgodClient("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "http://localhost:4001")
#    pprint.pprint(client.compile( core.get_sig_tmpl()))

    with open("sig.tmpl.teal", "w") as f:
        f.write(core.get_sig_tmpl())

