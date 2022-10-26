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


        self.map = {"name":"lsig.teal","version":6,"source":"","bytecode":"BiABAYEASIAASDEQgQYSRDEZIhJEMRiBABJEMSCAABJEMQGBABJEMQkyAxJEMRUyAxJEIg==",
                    "template_labels":{
                        "TMPL_ADDR_IDX":{"source_line":3,"position":5,"bytes":False},
                        "TMPL_EMITTER_ID":{"source_line":5,"position":8,"bytes":True},
                        "TMPL_APP_ID":{"source_line":16,"position":24,"bytes":False},
                        "TMPL_APP_ADDRESS":{"source_line":20,"position":30,"bytes":True}
                    },
                    "label_map":{},"line_map":[0,1,4,6,7,9,10,12,14,15,16,18,19,20,21,23,25,26,27,29,31,32,33,35,37,38,39,41,43,44,45,47,49,50,51]
        }


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

        #pprint.pprint({"values": values, "contract": bytes(contract).hex()})

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

    def get_bytecode_raw(self, idx: int):
        start = 0
        if idx > 0:
            start = list(self.sorted.values())[idx - 1]["position"] + 1

        stop = len(self.src)
        if idx < len(self.sorted):
            stop = list(self.sorted.values())[idx]["position"]

        chunk = self.src[start:stop]
        return chunk

    def get_sig_tmpl(self):
        def sig_tmpl():
            admin_app_id = ScratchVar()
            admin_address= ScratchVar()

            return Seq(
                # Just putting adding this as a tmpl var to make the address unique and deterministic
                # We don't actually care what the value is, pop it
                Pop(Tmpl.Int("TMPL_ADDR_IDX")),
                Pop(Tmpl.Bytes("TMPL_EMITTER_ID")),
                
                Assert(Txn.type_enum() == TxnType.ApplicationCall),
                Assert(Txn.on_completion() == OnComplete.OptIn),
                Assert(Txn.application_id() == Tmpl.Int("TMPL_APP_ID")),
                Assert(Txn.rekey_to() == Tmpl.Bytes("TMPL_APP_ADDRESS")),

                Assert(Txn.fee() == Int(0)),
                Assert(Txn.close_remainder_to() == Global.zero_address()),
                Assert(Txn.asset_close_to() == Global.zero_address()),
                
                Approve()
            )
        
        return compileTeal(sig_tmpl(), mode=Mode.Signature, version=6, assembleConstants=True)

if __name__ == '__main__':
    core = TmplSig("sig")

    if len(sys.argv) == 1:
        file_name = "sig.tmpl.teal"
    else:
        file_name = sys.argv[1]
    
    with open(file_name, "w") as f:
        f.write(core.get_sig_tmpl())
