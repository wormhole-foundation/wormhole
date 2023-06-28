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
        return int.from_bytes(b64decode(txn.innerTxns[0]["logs"][0]), "big")

    def getVAA(self, client, sender, sid, app):
        if sid == None:
            raise Exception("getVAA called with a sid of None")

        saddr = get_application_address(app)

        # SOOO, we send a nop txn through to push the block forward
        # one

        # This is ONLY needed on a local net...  the indexer will sit
        # on the last block for 30 to 60 seconds... we don't want this
        # log in prod since it is wasteful of gas

        if (self.INDEXER_ROUND > 512):  # until they fix it
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

        while True:
            nexttoken = ""
            while True:
                response = self.myindexer.search_transactions( min_round=self.INDEXER_ROUND, note_prefix=self.NOTE_PREFIX, next_page=nexttoken)
#                pprint.pprint(response)
                for x in response["transactions"]:
#                    pprint.pprint(x)
                    for y in x["inner-txns"]:
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
#                            print(str(seq) + " != " + str(sid))
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

        a = transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=self.tokenid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"attestToken", asset_id],
            foreign_apps = [self.coreid],
            foreign_assets = [asset_id],
            accounts=[emitter_addr, creator, c["address"]],
            sp=sp
        )

        a.fee = a.fee * 2

        txns.append(a)

        resp = self.sendTxn(client, sender, txns, True)

        # Point us at the correct round
        self.INDEXER_ROUND = resp.confirmedRound

#        print(encode_address(resp.__dict__["logs"][0]))
#        print(encode_address(resp.__dict__["logs"][1]))
        return self.parseSeqFromLog(resp)

    def transferAsset(self, client, sender, asset_id, quantity, receiver, chain, fee, payload = None):
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

#        pprint.pprint(args)

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

#        pprint.pprint(resp.__dict__)
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


#        vaa = self.parseVAA(bytes.fromhex("01000000011300c412b9e5b304bde8f8633a41568991ca56b7c11a925847f0059e95010ec5241b761719f12d3f4a79d1515e08152b2e8584cd1e8217dd7743c2bf863b78b2bf040001aebade2f601a4e9083585b1bb5f98d421f116e0393f525b95d51afbe69051587531771dc127a5e9d7b74662bb7ac378d44181522dc748b1b0cbfe1b1de6ed39d01024b4e9fc86ac64aaeef84ea14e4265c3c186042a3ae9ab2933bf06c0cbf326b3c2b89e7d9854fc5204a447bd202592a72d1d6db3d007bef9fea0e35953afbd9f1010342e4446ac94545a0447851eda5d5e3b8c97c6f4ef338977562cd4ecbee2b8fea42d536d7655c28a7f7fb2ff5fc8e5775e892d853c9b2e4969f9ce054ede801700104af0d783996ccfd31d6fc6f86b634288cd2f9cc29695cfcbf12d915c1b9c383dc792c7abbe8126cd917fb8658a8de843d64171122db182453584c0c330e8889730105f34d45ec63ec0a0c4535303fd9c83a0fad6b0a112b27306a288c1b46f2a78399754536ecb07f1ab6c32d92ed50b11fef3668b23d5c1ca010ec4c924441367eac0006566671ff859eec8429874ba9e07dd107b22859cf5029928bebec6eb73cdca6752f91bb252bca76cb15ede1121a84a9a54dad126f50f282a47f7d30880ef86a3900076d0d1241e1fc9d039810a8aebd7cab863017c9420eb67f4578577c5ec4d37162723dcd6213ff6895f280a88ba70de1a5b9257fe2937cbdea007e84886abc46dd0108b24dcddaae10f5e12b7085a0c3885a050640af17ba265a448102401854183e9f3ae9a14cad1af64eb57c6f145c6f709d7ed6bb8712a6b315dc2780c9eb42812e0109df696bf506dfcd8fce57968a84d5f773706b117fad31f86bbb089ede77d71a6e54b7729f79a82e7d6e4a6797380796fbcb9ba9428e8fcdf0400515f8205b31c5010a90a03c76fdec510712b2a6ee52cc0b6df5c921437896756f34b3782aa486eb5b5d02df783664257539233502ec25bbda7dd754afc139823da8a43c0d3c91c279000b33549edd8353c4d577cb273b88b545ae547ad01e85161a4fbbbb371cff453d6311c787254e2852c3b874ea60c67d40efc3ee3f24b51bc3fe95cc0a873e8a3fb6000ce2e206214ae2b4b048857f061ed3cf8cef060c67a85ad863f266145238c5d2a85e38b4eb9b3be4d33f502df4c45762504eb43a6bf78f01363d1399b67c354df8000d2d362d64a2e3d1583e1299238829cc11d81e9b9820121c0a2eb91d542aa54c993861e8225bc3e8d028dc128d284118703a4ec69144d69402efd72a29bb9f6b8f000e6bf56fa3ae6303f495f1379b450eb52580d7d9098dd909762e6186d19e06480d2bba8f06602dbd6d3d5deac7080fc2e61bd1be97e442b63435c91fa72b33534c000fad870b47c86f6997286bd4def4bacc5a8abbfef3f730f62183c638131004ea2f706ab73ebfe8f4879bf54f580444acec212e96e41abaf4acfc3383f05478e528001089599974feaab33862cd881af13f1645079bd2fa2ff07ca744674c8556aaf97c5c9c90df332d5b4ad1428776b68612f0b1ecb98c2ebc83f44f42426f180062cd00116aa93eecb4d528afaa07b72484acd5b79ad20e9ad8e55ce37cb9138b4c12a8eb3d10fa7d932b06ac441905e0226d3420101971a72c5488e4bfef222de8c3acd1011203a3e3d8ec938ffbc3a27d8caf50fc925bd25bd286d5ad6077dffd7e205ce0806e166b661d502f8c49acf88d42fde20e6015830d5517a0bfd40f79963ded4d2d006227697a000f68690008a0ae83030f1423aa97121527f65bbbb97925b43b95231bb0478fd650a057cc4b00000000000000072003000000000000000000000000000000000000000000000000000000000007a12000000000000000000000000000000000000000000000000000000000000000000008cf9ee7255420a50c55ef35d4bdcdd8048dee5c3c1333ecd97aff98869ea280780008000000000000000000000000000000000000000000000000000000000007a1206869206d6f6d"))
#        pprint.pprint(vaa)
#        sys.exit(0)

        gt = GenTest(False)
        self.gt = gt

        client = self.getAlgodClient()

        print("Generating the foundation account...")
        foundation = self.getTemporaryAccount(client)
        player = self.getTemporaryAccount(client)
        player2 = self.getTemporaryAccount(client)
        player3 = self.getTemporaryAccount(client)

        self.coreid = 1004
        print("coreid = " + str(self.coreid))

        self.tokenid = 1006
        print("token bridge " + str(self.tokenid) + " address " + get_application_address(self.tokenid))

        self.testid = self.createTestApp(client, player2)
        print("testid " + str(self.testid) + " address " + get_application_address(self.testid))

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

#        pprint.pprint(self.getBalances(client, player.getAddress()))
#        pprint.pprint(self.getBalances(client, player2.getAddress()))
#        pprint.pprint(self.getBalances(client, player3.getAddress()))
#
#        print("Lets transfer that asset to one of our other accounts... first lets create the vaa")
#        # paul - transferFromAlgorand
#        sid = self.transferAsset(client, player2, self.testasset, 100, player3.getAddress(), 8, 0)
#        print("... track down the generated VAA")
#        vaa = self.getVAA(client, player, sid, self.tokenid)
#        print(".. and lets pass that to player3")
#        self.submitVAA(bytes.fromhex(vaa), client, player3)
#
#        pprint.pprint(self.getBalances(client, player.getAddress()))
#        pprint.pprint(self.getBalances(client, player2.getAddress()))
#        pprint.pprint(self.getBalances(client, player3.getAddress()))
#
#        # Lets split it into two parts... the payload and the fee
#        print("Lets split it into two parts... the payload and the fee")
#        sid = self.transferAsset(client, player2, self.testasset, 1000, player3.getAddress(), 8, 500)
#        print("... track down the generated VAA")
#        vaa = self.getVAA(client, player, sid, self.tokenid)
#        print(".. and lets pass that to player3 with fees being passed to player acting as a relayer")
#        self.submitVAA(bytes.fromhex(vaa), client, player)
#
#        pprint.pprint(self.getBalances(client, player.getAddress()))
#        pprint.pprint(self.getBalances(client, player2.getAddress()))
#        pprint.pprint(self.getBalances(client, player3.getAddress()))
#
#        # Now it gets tricky, lets create a virgin account...
#        pk, addr  = account.generate_account()
#        emptyAccount = Account(pk)
#
#        print("How much is in the empty account? (" + addr + ")")
#        pprint.pprint(self.getBalances(client, emptyAccount.getAddress()))
#
#        # paul - transferFromAlgorand
#        print("Lets transfer algo this time.... first lets create the vaa")
#        sid = self.transferAsset(client, player2, 0, 1000000, emptyAccount.getAddress(), 8, 0)
#        print("... track down the generated VAA")
#        vaa = self.getVAA(client, player, sid, self.tokenid)
##        pprint.pprint(vaa)
#        print(".. and lets pass that to the empty account.. but use somebody else to relay since we cannot pay for it")
#
#        # paul - redeemOnAlgorand
#        self.submitVAA(bytes.fromhex(vaa), client, player)
#
#        print("=================================================")
#
#        print("How much is in the source account now?")
#        pprint.pprint(self.getBalances(client, player2.getAddress()))
#
#        print("How much is in the empty account now?")
#        pprint.pprint(self.getBalances(client, emptyAccount.getAddress()))
#
#        print("How much is in the player3 account now?")
#        pprint.pprint(self.getBalances(client, player3.getAddress()))
#
#        print("Lets transfer more algo.. splut 50/50 with the relayer.. going to player3")
#        sid = self.transferAsset(client, player2, 0, 1000000, player3.getAddress(), 8, 500000)
#        print("... track down the generated VAA")
#        vaa = self.getVAA(client, player, sid, self.tokenid)
#        print(".. and lets pass that to player3.. but use the previously empty account to relay it")
#        self.submitVAA(bytes.fromhex(vaa), client, emptyAccount)
#
#        print("How much is in the source account now?")
#        pprint.pprint(self.getBalances(client, player2.getAddress()))
#
#        print("How much is in the empty account now?")
#        pprint.pprint(self.getBalances(client, emptyAccount.getAddress()))
#
#        print("How much is in the player3 account now?")
#        pprint.pprint(self.getBalances(client, player3.getAddress()))
#
#        print("How about a payload3")
#        sid = self.transferAsset(client, player2, 0, 100, player3.getAddress(), 8, 0, b'hi mom')
#        print("... track down the generated VAA")
#        vaa = self.getVAA(client, player, sid, self.tokenid)
#
#        print(".. and lets pass that to the wrong account")
#        try:
#            self.submitVAA(bytes.fromhex(vaa), client, emptyAccount)
#        except:
#            print("Exception thrown... nice")
#
#        print(".. and lets pass that to the right account")
#        self.submitVAA(bytes.fromhex(vaa), client, player3)

#        print("player account: " + player.getAddress())
#        pprint.pprint(client.account_info(player.getAddress()))

#        print("player2 account: " + player2.getAddress())
#        pprint.pprint(client.account_info(player2.getAddress()))

#        print("foundation account: " + foundation.getAddress())
#        pprint.pprint(client.account_info(foundation.getAddress()))
#
#        print("core app: " + get_application_address(self.coreid))
#        pprint.pprint(client.account_info(get_application_address(self.coreid))),
#
#        print("token app: " + get_application_address(self.tokenid))
#        pprint.pprint(client.account_info(get_application_address(self.tokenid))),
#
#        print("asset app: " + chain_addr)
#        pprint.pprint(client.account_info(chain_addr))

core = AlgoTest()
core.simple_test()
