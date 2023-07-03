# python3 -m pip install pycryptodomex uvarint pyteal web3 coincurve

import sys
sys.path.append("..")

from admin import PortalCore, Account
from gentest import GenTest
from base64 import b64decode

from typing import List, Tuple, Dict, Any, Optional, Union
import base64
import random
import time
import hashlib
import uuid
import json

from algosdk.v2client.algod import AlgodClient
from algosdk.kmd import KMDClient
from algosdk import account, mnemonic
from algosdk.encoding import decode_address, encode_address
from algosdk.future import transaction
from pyteal import compileTeal, Mode, Expr
from pyteal import *
from algosdk.logic import get_application_address
from vaa_verify import get_vaa_verify

from algosdk.future.transaction import LogicSig

from test_contract import get_test_app

from algosdk.v2client import indexer

import pprint

class AlgoTest(PortalCore):
    def __init__(self) -> None:
        super().__init__()

    def getBalances(self, client: AlgodClient, account: str) -> Dict[int, int]:
        balances: Dict[int, int] = dict()
    
        accountInfo = client.account_info(account)
    
        # set key 0 to Algo balance
        balances[0] = accountInfo["amount"]
    
        assets: List[Dict[str, Any]] = accountInfo.get("assets", [])
        for assetHolding in assets:
            assetID = assetHolding["asset-id"]
            amount = assetHolding["amount"]
            balances[assetID] = amount
    
        return balances

    def createTestApp(
        self,
        client: AlgodClient,
        sender: Account,
    ) -> int:
        approval, clear = get_test_app(client)

        globalSchema = transaction.StateSchema(num_uints=4, num_byte_slices=30)
        localSchema = transaction.StateSchema(num_uints=0, num_byte_slices=16)
    
        app_args = []

        txn = transaction.ApplicationCreateTxn(
            sender=sender.getAddress(),
            on_complete=transaction.OnComplete.NoOpOC,
            approval_program=b64decode(approval["result"]),
            clear_program=b64decode(clear["result"]),
            global_schema=globalSchema,
            local_schema=localSchema,
            app_args=app_args,
            sp=client.suggested_params(),
        )
    
        signedTxn = txn.sign(sender.getPrivateKey())
    
        client.send_transaction(signedTxn)
    
        response = self.waitForTransaction(client, signedTxn.get_txid())
        assert response.applicationIndex is not None and response.applicationIndex > 0

        txn = transaction.PaymentTxn(sender = sender.getAddress(), sp = client.suggested_params(), 
                                     receiver = get_application_address(response.applicationIndex), amt = 300000)
        signedTxn = txn.sign(sender.getPrivateKey())
        client.send_transaction(signedTxn)

        return response.applicationIndex

    def parseSeqFromLog(self, txn):
        try:
            return int.from_bytes(b64decode(txn.innerTxns[-1]["logs"][0]), "big")
        except Exception as err:
            pprint.pprint(txn.__dict__)
            raise

    def getVAA(self, client, sender, sid, app):
        if sid == None:
            raise Exception("getVAA called with a sid of None")

        saddr = get_application_address(app)

        # SOOO, we send a nop txn through to push the block forward
        # one

        # This is ONLY needed on a local net...  the indexer will sit
        # on the last block for 30 to 60 seconds... we don't want this
        # log in prod since it is wasteful of gas

        if (self.INDEXER_ROUND > 512 and not self.args.testnet):  # until they fix it
            print("indexer is broken in local net... stop/clean/restart the sandbox")
            sys.exit(0)

        txns = []

        txns.append(
            transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=self.tokenid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"nop"],
                sp=client.suggested_params(),
            )
        )
        self.sendTxn(client, sender, txns, False)

        if self.myindexer == None:
            print("indexer address: " + self.INDEXER_ADDRESS)
            self.myindexer = indexer.IndexerClient(indexer_token=self.INDEXER_TOKEN, indexer_address=self.INDEXER_ADDRESS)

        while True:
            nexttoken = ""
            while True:
                response = self.myindexer.search_transactions( min_round=self.INDEXER_ROUND, note_prefix=self.NOTE_PREFIX, next_page=nexttoken)
#                pprint.pprint(response)
                for x in response["transactions"]:
#                    pprint.pprint(x)
                    for y in x["inner-txns"]:
                        if "application-transaction" not in y:
                            continue
                        if y["application-transaction"]["application-id"] != self.coreid:
                            continue
                        if len(y["logs"]) == 0:
                            continue
                        args = y["application-transaction"]["application-args"]
                        if len(args) < 2:
                            continue
                        if base64.b64decode(args[0]) != b'publishMessage':
                            continue
                        seq = int.from_bytes(base64.b64decode(y["logs"][0]), "big")
                        if seq != sid:
                            continue
                        if y["sender"] != saddr:
                            continue;
                        emitter = decode_address(y["sender"])
                        payload = base64.b64decode(args[1])
#                        pprint.pprint([seq, y["sender"], payload.hex()])
#                        sys.exit(0)
                        return self.gt.genVaa(emitter, seq, payload)

                if 'next-token' in response:
                    nexttoken = response['next-token']
                else:
                    self.INDEXER_ROUND = response['current-round'] + 1
                    break
            time.sleep(1)
        
    def publishMessage(self, client, sender, vaa, appid):
        aa = decode_address(get_application_address(appid)).hex()
        emitter_addr = self.optin(client, sender, self.coreid, 0, aa)

        txns = []
        sp = client.suggested_params()

        a = transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=appid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"test1", vaa, self.coreid],
            foreign_apps = [self.coreid],
            accounts=[emitter_addr],
            sp=sp
        )

        a.fee = a.fee * 2

        txns.append(a)

        resp = self.sendTxn(client, sender, txns, True)

        self.INDEXER_ROUND = resp.confirmedRound

        return self.parseSeqFromLog(resp)

    def createTestAsset(self, client, sender):
        txns = []

        a = transaction.PaymentTxn(
            sender = sender.getAddress(), 
            sp = client.suggested_params(), 
            receiver = get_application_address(self.testid), 
            amt = 300000
        )

        txns.append(a)

        sp = client.suggested_params()
        a = transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=self.testid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"setup"],
            sp=sp
        )

        a.fee = a.fee * 2

        txns.append(a)
        transaction.assign_group_id(txns)

        grp = []
        pk = sender.getPrivateKey()
        for t in txns:
            grp.append(t.sign(pk))

        client.send_transactions(grp)
        resp = self.waitForTransaction(client, grp[-1].get_txid())
        
        aid = int.from_bytes(resp.__dict__["logs"][0], "big")

        print("Opting " + sender.getAddress() + " into " + str(aid))
        self.asset_optin(client, sender, aid, sender.getAddress())

        txns = []
        a = transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=self.testid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"mint"],
            foreign_assets = [aid],
            sp=sp
        )

        a.fee = a.fee * 2

        txns.append(a)

        resp = self.sendTxn(client, sender, txns, True)

#        self.INDEXER_ROUND = resp.confirmedRound

        return aid

    def getCreator(self, client, sender, asset_id):
        return client.asset_info(asset_id)["params"]["creator"]

    def testAttest(self, client, sender, asset_id):
        taddr = get_application_address(self.tokenid)
        aa = decode_address(taddr).hex()
        emitter_addr = self.optin(client, sender, self.coreid, 0, aa)

        creator = self.getCreator(client, sender, asset_id)
        c = client.account_info(creator)
        wormhole = c.get("auth-addr") == taddr

        if not wormhole:
            creator = self.optin(client, sender, self.tokenid, asset_id, b"native".hex())

        txns = []
        sp = client.suggested_params()

        txns.append(transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=self.tokenid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"nop"],
            sp=sp
        ))

        mfee = self.getMessageFee()
        if (mfee > 0):
            txns.append(transaction.PaymentTxn(sender = sender.getAddress(), sp = sp, receiver = get_application_address(self.tokenid), amt = mfee))

        a = transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=self.tokenid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"attestToken", asset_id],
            foreign_apps = [self.coreid],
            foreign_assets = [asset_id],
            accounts=[emitter_addr, creator, c["address"], get_application_address(self.coreid)],
            sp=sp
        )

        if (mfee > 0):
            a.fee = a.fee * 3
        else:
            a.fee = a.fee * 2

        txns.append(a)

        resp = self.sendTxn(client, sender, txns, True)

        # Point us at the correct round
        self.INDEXER_ROUND = resp.confirmedRound

#        print(encode_address(resp.__dict__["logs"][0]))
#        print(encode_address(resp.__dict__["logs"][1]))
#        pprint.pprint(resp.__dict__)
        return self.parseSeqFromLog(resp)

    def transferAsset(self, client, sender, asset_id, quantity, receiver, chain, fee, payload = None):
#        pprint.pprint(["transferAsset", asset_id, quantity, receiver, chain, fee])

        taddr = get_application_address(self.tokenid)
        aa = decode_address(taddr).hex()
        emitter_addr = self.optin(client, sender, self.coreid, 0, aa)

        # asset_id 0 is ALGO

        if asset_id == 0:
            wormhole = False
        else:
            creator = self.getCreator(client, sender, asset_id)
            c = client.account_info(creator)
            wormhole = c.get("auth-addr") == taddr

        txns = []


        mfee = self.getMessageFee()
        if (mfee > 0):
            txns.append(transaction.PaymentTxn(sender = sender.getAddress(), sp = sp, receiver = get_application_address(self.tokenid), amt = mfee))

        if not wormhole:
            creator = self.optin(client, sender, self.tokenid, asset_id, b"native".hex())
            print("non wormhole account " + creator)

        sp = client.suggested_params()

        if (asset_id != 0) and (not self.asset_optin_check(client, sender, asset_id, creator)):
            print("Looks like we need to optin")

            txns.append(
                transaction.PaymentTxn(
                    sender=sender.getAddress(),
                    receiver=creator,
                    amt=100000,
                    sp=sp
                )
            )

            # The tokenid app needs to do the optin since it has signature authority
            a = transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=self.tokenid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"optin", asset_id],
                foreign_assets = [asset_id],
                accounts=[creator],
                sp=sp
            )

            a.fee = a.fee * 2
            txns.append(a)
            self.sendTxn(client, sender, txns, False)
            txns = []

        txns.insert(0, 
            transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=self.tokenid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"nop"],
                sp=client.suggested_params(),
            )
        )

        if asset_id == 0:
            print("asset_id == 0")
            txns.append(transaction.PaymentTxn(
                sender=sender.getAddress(),
                receiver=creator,
                amt=quantity,
                sp=sp,
            ))
            accounts=[emitter_addr, creator, creator]
        else:
            print("asset_id != 0")
            txns.append(
                transaction.AssetTransferTxn(
                    sender = sender.getAddress(), 
                    sp = sp, 
                    receiver = creator,
                    amt = quantity,
                    index = asset_id
                ))
            accounts=[emitter_addr, creator, c["address"]]

        args = [b"sendTransfer", asset_id, quantity, decode_address(receiver), chain, fee]
        if None != payload:
            args.append(payload)

        #pprint.pprint(args)

#        print(self.tokenid)
        a = transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=self.tokenid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=args,
            foreign_apps = [self.coreid],
            foreign_assets = [asset_id],
            accounts=accounts,
            sp=sp
        )

        a.fee = a.fee * 2

        txns.append(a)

        resp = self.sendTxn(client, sender, txns, True)

        self.INDEXER_ROUND = resp.confirmedRound

#        pprint.pprint([self.coreid, self.tokenid, resp.__dict__,
#                       int.from_bytes(resp.__dict__["logs"][1], "big"),
#                       int.from_bytes(resp.__dict__["logs"][2], "big"),
#                       int.from_bytes(resp.__dict__["logs"][3], "big"),
#                       int.from_bytes(resp.__dict__["logs"][4], "big"),
#                       int.from_bytes(resp.__dict__["logs"][5], "big")
#                       ])
#        print(encode_address(resp.__dict__["logs"][0]))
#        print(encode_address(resp.__dict__["logs"][1]))
        return self.parseSeqFromLog(resp)

    def asset_optin_check(self, client, sender, asset, receiver):
        if receiver not in self.asset_cache:
            self.asset_cache[receiver] = {}

        if asset in self.asset_cache[receiver]:
            return True

        ai = client.account_info(receiver)
        if "assets" in ai:
            for x in ai["assets"]:
                if x["asset-id"] == asset:
                    self.asset_cache[receiver][asset] = True
                    return True

        return False

    def asset_optin(self, client, sender, asset, receiver):
        if self.asset_optin_check(client, sender, asset, receiver):
            return

        pprint.pprint(["asset_optin", asset, receiver])

        sp = client.suggested_params()
        optin_txn = transaction.AssetTransferTxn(
            sender = sender.getAddress(), 
            sp = sp, 
            receiver = receiver, 
            amt = 0, 
            index = asset
        )

        transaction.assign_group_id([optin_txn])
        signed_optin = optin_txn.sign(sender.getPrivateKey())
        client.send_transactions([signed_optin])
        resp = self.waitForTransaction(client, signed_optin.get_txid())
        assert self.asset_optin_check(client, sender, asset, receiver), "The optin failed"
        print("woah! optin succeeded")

    def simple_test(self):
#        q = bytes.fromhex(gt.genAssetMeta(gt.guardianPrivKeys, 1, 1, 1, bytes.fromhex("4523c3F29447d1f32AEa95BEBD00383c4640F1b4"), 1, 8, b"USDC", b"CircleCoin"))
#        pprint.pprint(self.parseVAA(q))
#        sys.exit(0)


#        vaa = self.parseVAA(bytes.fromhex("01000000010100e1232697de3681d67ca0c46fbbc9ea5d282c473daae8fda2b23145e7b7167f9a35888acf80ed9d091af3069108c25324a22d8665241db884dda53ca53a8212d100625436600000000100020000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585000000000000000120010000000000000000000000000000000000000000000000000000000005f5e1000000000000000000000000000000000000000000000000004523c3F29447d1f32AEa95BEBD00383c4640F1b400020000000000000000000000000000000000000000000000000000aabbcc00080000000000000000000000000000000000000000000000000000000000000000"))
#        pprint.pprint(vaa)
#        sys.exit(0)

        gt = GenTest(True)
        self.gt = gt

        self.setup_args()

        if self.args.testnet:
            self.testnet()

        client = self.client = self.getAlgodClient()

        self.genTeal()

        self.vaa_verify = self.client.compile(get_vaa_verify())
        self.vaa_verify["lsig"] = LogicSig(base64.b64decode(self.vaa_verify["result"]))

        vaaLogs = []

        args = self.args

        if self.args.mnemonic:
            self.foundation = Account.FromMnemonic(self.args.mnemonic)

        if self.foundation == None:
            print("Generating the foundation account...")
            self.foundation = self.getTemporaryAccount(self.client)

        if self.foundation == None:
            print("We dont have a account?  ")
            sys.exit(0)

        foundation = self.foundation

        seq = int(time.time())

        self.coreid = 1004
        self.tokenid = 1006


        player = self.getTemporaryAccount(client)
        print("token bridge " + str(self.tokenid) + " address " + get_application_address(self.tokenid))

        player2 = self.getTemporaryAccount(client)
        player3 = self.getTemporaryAccount(client)

        # This creates a asset by using another app... you can also just creat the asset from the client sdk like we do in the typescript test
        self.testid = self.createTestApp(client, player2)

        print("Lets create a brand new non-wormhole asset and try to attest and send it out")
        self.testasset = self.createTestAsset(client, player2)
        
        print("test asset id: " + str(self.testasset))

        print("Lets try to create an attest for a non-wormhole thing with a huge number of decimals")
        # paul - attestFromAlgorand
        sid = self.testAttest(client, player2, self.testasset)
        print("... track down the generated VAA")
        vaa = self.getVAA(client, player, sid, self.tokenid)
        v = self.parseVAA(bytes.fromhex(vaa))
        print("We got a " + v["Meta"])

if __name__ == "__main__":
    core = AlgoTest()
    core.simple_test()
