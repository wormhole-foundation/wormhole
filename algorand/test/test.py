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
import algosdk
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
                response = self.myindexer.search_transactions( min_round=self.INDEXER_ROUND, next_page=nexttoken)
#                pprint.pprint(response)
                for x in response["transactions"]:
#                    pprint.pprint(x)
                    if 'inner-txns' not in x:
                        continue

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

        if asset_id != 0:
            creator = self.getCreator(client, sender, asset_id)
            c = client.account_info(creator)
            wormhole = c.get("auth-addr") == taddr
        else:
            c = None
            wormhole = False

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

        accts = [emitter_addr, creator, get_application_address(self.coreid)]
        if c != None:
            accts.append(c["address"])

        a = transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=self.tokenid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"attestToken", asset_id],
            foreign_apps = [self.coreid],
            foreign_assets = [asset_id],
            accounts=accts,
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

    def transferFromAlgorand(self, client, sender, asset_id, quantity, receiver, chain, fee, payload = None):
#        pprint.pprint(["transferFromAlgorand", asset_id, quantity, receiver, chain, fee])

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

        args = [b"sendTransfer", asset_id, quantity, receiver, chain, fee]
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

        #vaa = self.parseVAA(bytes.fromhex("01000000000100ddc6993585b909c3e861830244122e0daf45101663942484aa56ee2b51fa3ff016411102f993935d428a5aa0c3ace74facd60822435893b74b24fadde0fbad49006277c3fe0000000000088edf5b0e108c3a1a0a4b704cc89591f2ad8d50df24e991567e640ed720a94be200000000000000060003000000000000000000000000000000000000000000000000000000000000006400000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000800080000000000000000000000000000000000000000000000000000000000000000ff"))
        #pprint.pprint(vaa)
        #sys.exit(0)

#        q = bytes.fromhex(gt.genAssetMeta(gt.guardianPrivKeys, 1, 1, 1, bytes.fromhex("4523c3F29447d1f32AEa95BEBD00383c4640F1b4"), 1, 8, b"USDC", b"CircleCoin"))
#        pprint.pprint(self.parseVAA(q))
#        sys.exit(0)


#        vaa = self.parseVAA(bytes.fromhex("0100000001010001ca2fbf60ac6227d47dda4fe2e7bccc087f27d22170a212b9800da5b4cbf0d64c52deb2f65ce58be2267bf5b366437c267b5c7b795cd6cea1ac2fee8a1db3ad006225f801000000010001000000000000000000000000000000000000000000000000000000000000000400000000000000012000000000000000000000000000000000000000000000000000000000436f72650200000000000001beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"))
#        pprint.pprint(vaa)
#        vaa = self.parseVAA(bytes.fromhex("01000000010100c22ce0a3c995fca993cb0e91af74d745b6ec1a04b3adf0bb3e432746b3e2ab5e635b65d34d5148726cac10e84bf5932a7f21b9545c362bd512617aa980e0fbf40062607566000000010001000000000000000000000000000000000000000000000000000000000000000400000000000000012000000000000000000000000000000000000000000000000000000000436f72650200000000000101beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"))
#        pprint.pprint(vaa)
#        sys.exit(0)


        self.setup_args()

        gt = GenTest(self.args.bigset)
        self.gt = gt

        if self.args.testnet:
            self.testnet()
        else:
            self.devnet = True

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
            sys.exit(1)

        foundation = self.foundation

        seq = int(time.time())

        print("Creating the PortalCore app")
        self.coreid = self.createPortalCoreApp(client=client, sender=foundation)
        print("coreid = " + str(self.coreid) + " " + get_application_address(self.coreid))

        print("bootstrapping the guardian set...")
        bootVAA = bytes.fromhex(gt.genGuardianSetUpgrade(gt.guardianPrivKeys, 1, 1, seq, seq))

        self.bootGuardians(bootVAA, client, foundation, self.coreid)

        seq += 1

        print("grabbing a untrusted account")
        player = self.getTemporaryAccount(client)
        print(player.getAddress())
        print("")

        bal = self.getBalances(client, player.getAddress())
        pprint.pprint(bal)

        print("upgrading the the guardian set using untrusted account...")
        upgradeVAA = bytes.fromhex(gt.genGuardianSetUpgrade(gt.guardianPrivKeys, 1, 2, seq, seq))
        vaaLogs.append(["guardianUpgrade", upgradeVAA.hex()])
        self.submitVAA(upgradeVAA, client, player, self.coreid)

        bal = self.getBalances(client, player.getAddress())
        pprint.pprint(bal)

        seq += 1

        print("Create the token bridge")
        self.tokenid = self.createTokenBridgeApp(client, foundation)
        print("token bridge " + str(self.tokenid) + " address " + get_application_address(self.tokenid))

        ret = self.devnetUpgradeVAA()
#        pprint.pprint(ret)
        print("Submitting core")
        self.submitVAA(bytes.fromhex(ret[0]), self.client, foundation, self.coreid)
        print("Submitting token")
        self.submitVAA(bytes.fromhex(ret[1]), self.client, foundation, self.tokenid)

        print("successfully sent upgrade requests")

        for r in range(1, 6):
            print("Registering chain " + str(r))
            v = gt.genRegisterChain(gt.guardianPrivKeys, 2, seq, seq, r)
            vaa = bytes.fromhex(v)
#            pprint.pprint((v, self.parseVAA(vaa)))
            if r == 2:
                vaaLogs.append(["registerChain", v])
            self.submitVAA(vaa, client, player, self.tokenid)
            seq += 1

            bal = self.getBalances(client, player.getAddress())
            pprint.pprint(bal)

        print("Create a asset")
        attestVAA = bytes.fromhex(gt.genAssetMeta(gt.guardianPrivKeys, 2, seq, seq, bytes.fromhex("4523c3F29447d1f32AEa95BEBD00383c4640F1b4"), 1, 8, b"USDC", b"CircleCoin"))
        # paul - createWrappedOnAlgorand
        vaaLogs.append(["createWrappedOnAlgorand", attestVAA.hex()])
        self.submitVAA(attestVAA, client, player, self.tokenid)
        seq += 1

        p = self.parseVAA(attestVAA)
        chain_addr = self.optin(client, player, self.tokenid, p["FromChain"], p["Contract"])

        print("Create the same asset " + str(seq))
        # paul - updateWrappedOnAlgorand
        attestVAA = bytes.fromhex(gt.genAssetMeta(gt.guardianPrivKeys, 2, seq, seq, bytes.fromhex("4523c3F29447d1f32AEa95BEBD00383c4640F1b4"), 1, 8, b"USD2C", b"Circle2Coin"))
        self.submitVAA(attestVAA, client, player, self.tokenid)
        seq += 1

        print("Transfer the asset " + str(seq))
        transferVAA = bytes.fromhex(gt.genTransfer(gt.guardianPrivKeys, 1, 1, 1, 1, bytes.fromhex("4523c3F29447d1f32AEa95BEBD00383c4640F1b4"), 1, decode_address(player.getAddress()), 8, 0))
        # paul - redeemOnAlgorand
        vaaLogs.append(["redeemOnAlgorand", transferVAA.hex()])
        self.submitVAA(transferVAA, client, player, self.tokenid)
        seq += 1

        def double_submit_transfer_vaa_fails(seq):
            """
            Resend the same transaction we just send, changing only its nonce.
            This should fail _as long as the sequence number is not incremented_
            """

            # send a nice VAA to begin with. everything but these settings will be random
            # so we can be sure this works with many different VAAs -- as long as they are valid
            # non-valid vaas fail for other reasons
            vaa = bytearray.fromhex(gt.genRandomValidTransfer(
                signers=gt.guardianPrivKeys,
                guardianSet=1,
                seq=seq,
                # we set the max_amount, but the actual amount will be between zero and this value
                amount_max=self.getBalances(client, player.getAddress())[0], # 0 is the ALGO amount
                tokenAddress=bytes.fromhex("4523c3F29447d1f32AEa95BEBD00383c4640F1b4"),
                toAddress=decode_address(player.getAddress()),
            ))

            self.submitVAA(vaa, client, player, self.tokenid)

            # Let's make this even stronger: scramble the few bytes we can (len_signatures, signatures)
            # so the repeated one is still valid, but different from the first one.
            # NOTE: this will only be interesting if we are working with a big validator set,
            # don't even botters if it's not
            if len(gt.guardianKeys) > 1:
                current_signatures_amount = vaa[5]
                signatures_len = 66*current_signatures_amount
                signatures_offset = 6
                rest_offset = signatures_offset+signatures_len

                new_signature_amount = random.randint(int(len(gt.guardianKeys)*2/3)+1, current_signatures_amount)

                # construct a list of every siganture with its index
                signatures = vaa[signatures_offset:rest_offset]
                signatures = [signatures[i:i+66] for i in range(0, len(signatures), 66)]
                assert len(signatures) == current_signatures_amount

                # scramble the signatures so we get new bytes
                new_signatures = random.sample(signatures, k=new_signature_amount)
                assert len(new_signatures) == new_signature_amount
                new_signatures = b''.join(new_signatures)

                vaa[5] = new_signature_amount
                new_vaa = vaa[:6] + new_signatures + vaa[rest_offset:]
                assert(len(new_vaa) == len(vaa)-((current_signatures_amount-new_signature_amount)*66))
                vaa = new_vaa

            # now try again!
            try:
                self.submitVAA(vaa, client, player, self.tokenid)
            except algosdk.error.AlgodHTTPError as e:
                # should fail right at line 963
                if "opcodes=pushint 963" in str(e):
                    return True, vaa, None
                return False, vaa, e

            return False, vaa, None

        for _ in range(self.args.loops):
            result, vaa, err = double_submit_transfer_vaa_fails(seq)
            if err != None:
                assert False, f"!!! ERR: unepexted error. error:\n {err}\noffending vaa hex:\n{vaa.hex()}"

            assert result, f"!!! ERR: sending same VAA twice worked. offending vaa hex:\n{vaa.hex()}"
            seq+=1
        return

        def sending_vaa_version_not_one_fails(seq, version):
            vaa = bytearray.fromhex(gt.genRandomValidTransfer(
                signers=gt.guardianPrivKeys,
                guardianSet=1,
                seq=seq,
                tokenAddress=bytes.fromhex("4523c3F29447d1f32AEa95BEBD00383c4640F1b4"),
                toAddress=decode_address(player.getAddress()),
                amount_max=self.getBalances(client, player.getAddress())[0], # 0 is the ALGO amount
                ))

            # we know VAA is malleable in the first four fields:
            # version, guardian set index, len of signatures, signatures
            vaa[0] = version

            try:
                self.submitVAA(vaa, client, player, self.tokenid)
            except algosdk.error.AlgodHTTPError as e:
                # right at the beginning of checkForDuplicate()
                if "opcodes=pushint 919" in str(e):
                    return True, vaa, None
                return False, vaa, e

            return False, vaa, None

        # no need to increase _seq_ after this one as if everything went ok...
        # all VAAs should have been invalid!
        for _ in range(self.args.loops):
            version = random.randint(0, 255)

            if version == 1:
                continue

            ok, vaa, err = sending_vaa_version_not_one_fails(seq, version)
            if err != None:
                assert False, f"!!! ERR: unepexted error when testing version. error:\n {err}\noffending vaa hex:\n{vaa.hex()}"

            assert ok, f"!!! ERR: Invalid version worked. offending version: {version}. offending vaa:\n{vaa}"

        print("Create the test app we will use to torture ourselves using a new player")
        player2 = self.getTemporaryAccount(client)
        print("player2 address " + player2.getAddress())
        player3 = self.getTemporaryAccount(client)
        print("player3 address " + player3.getAddress())

        self.testid = self.createTestApp(client, player2)
        print("testid " + str(self.testid) + " address " + get_application_address(self.testid))

        print("Sending a message payload to the core contract")
        sid = self.publishMessage(client, player, b"you also suck", self.testid)
        self.publishMessage(client, player2, b"second suck", self.testid)
        self.publishMessage(client, player3, b"last message", self.testid)

        print("Lets create a brand new non-wormhole asset and try to attest and send it out")
        self.testasset = self.createTestAsset(client, player2)
        print("test asset id: " + str(self.testasset))
        
        print("Now lets create an attest of ALGO")
        sid = self.testAttest(client, player2, 0)
        vaa = self.getVAA(client, player, sid, self.tokenid)
        v = self.parseVAA(bytes.fromhex(vaa))
        print("We got a " + str(v["Meta"]))

        print("Lets try to create an attest for a non-wormhole thing with a huge number of decimals")
        # paul - attestFromAlgorand
        sid = self.testAttest(client, player2, self.testasset)
        print("... track down the generated VAA")
        vaa = self.getVAA(client, player, sid, self.tokenid)
        v = self.parseVAA(bytes.fromhex(vaa))
        print("We got a " + v["Meta"])

        pprint.pprint(self.getBalances(client, player.getAddress()))
        pprint.pprint(self.getBalances(client, player2.getAddress()))
        pprint.pprint(self.getBalances(client, player3.getAddress()))

        print("Lets transfer that asset to one of our other accounts... first lets create the vaa")
        # paul - transferFromAlgorand
        sid = self.transferFromAlgorand(client, player2, self.testasset, 100, decode_address(player3.getAddress()), 8, 0)
        print("... track down the generated VAA")
        vaa = self.getVAA(client, player, sid, self.tokenid)
        print(".. and lets pass that to player3")
        vaaLogs.append(["transferFromAlgorand", vaa])
        #pprint.pprint(vaaLogs)
        self.submitVAA(bytes.fromhex(vaa), client, player3, self.tokenid)

        pprint.pprint(["player", self.getBalances(client, player.getAddress())])
        pprint.pprint(["player2", self.getBalances(client, player2.getAddress())])
        pprint.pprint(["player3", self.getBalances(client, player3.getAddress())])

        # Lets split it into two parts... the payload and the fee
        print("Lets split it into two parts... the payload and the fee (400 should go to player, 600 should go to player3)")
        sid = self.transferFromAlgorand(client, player2, self.testasset, 1000, decode_address(player3.getAddress()), 8, 400)
        print("... track down the generated VAA")
        vaa = self.getVAA(client, player, sid, self.tokenid)
#        pprint.pprint(self.parseVAA(bytes.fromhex(vaa)))
        print(".. and lets pass that to player3 with fees being passed to player acting as a relayer (" + str(self.tokenid) + ")")
        self.submitVAA(bytes.fromhex(vaa), client, player, self.tokenid)

        pprint.pprint(["player", self.getBalances(client, player.getAddress())])
        pprint.pprint(["player2", self.getBalances(client, player2.getAddress())])
        pprint.pprint(["player3", self.getBalances(client, player3.getAddress())])

#        sys.exit(0)

        # Now it gets tricky, lets create a virgin account...
        pk, addr  = account.generate_account()
        emptyAccount = Account(pk)

        print("How much is in the empty account? (" + addr + ")")
        pprint.pprint(self.getBalances(client, emptyAccount.getAddress()))

        # paul - transferFromAlgorand
        print("Lets transfer algo this time.... first lets create the vaa")
        sid = self.transferFromAlgorand(client, player2, 0, 1000000, decode_address(emptyAccount.getAddress()), 8, 0)
        print("... track down the generated VAA")
        vaa = self.getVAA(client, player, sid, self.tokenid)
#        pprint.pprint(vaa)
        print(".. and lets pass that to the empty account.. but use somebody else to relay since we cannot pay for it")

        # paul - redeemOnAlgorand
        self.submitVAA(bytes.fromhex(vaa), client, player, self.tokenid)

        print("=================================================")

        print("How much is in the source account now?")
        pprint.pprint(self.getBalances(client, player2.getAddress()))

        print("How much is in the empty account now?")
        pprint.pprint(self.getBalances(client, emptyAccount.getAddress()))

        print("How much is in the player3 account now?")
        pprint.pprint(self.getBalances(client, player3.getAddress()))

        print("Lets transfer more algo.. split 40/60 with the relayer.. going to player3")
        sid = self.transferFromAlgorand(client, player2, 0, 1000000, decode_address(player3.getAddress()), 8, 400000)
        print("... track down the generated VAA")
        vaa = self.getVAA(client, player, sid, self.tokenid)
        print(".. and lets pass that to player3.. but use the previously empty account to relay it")
        self.submitVAA(bytes.fromhex(vaa), client, emptyAccount, self.tokenid)

        print("How much is in the source account now?")
        pprint.pprint(self.getBalances(client, player2.getAddress()))

        print("How much is in the empty account now?")
        pprint.pprint(self.getBalances(client, emptyAccount.getAddress()))

        print("How much is in the player3 account now?")
        pprint.pprint(self.getBalances(client, player3.getAddress()))

        print("How about a payload3: " + self.testid.to_bytes(32, "big").hex())
        sid = self.transferFromAlgorand(client, player2, 0, 100, self.testid.to_bytes(32, "big"), 8, 0, b'hi mom')
        print("... track down the generated VAA")
        vaa = self.getVAA(client, player, sid, self.tokenid)

        print("player address: " + decode_address(player2.getAddress()).hex())
        print("payload3 vaa: "+ vaa)
        pprint.pprint(self.parseVAA(bytes.fromhex(vaa)))

        print("testid balance before = ", self.getBalances(client, get_application_address(self.testid)))

        print(".. Lets let player3 relay it for us")
        self.submitVAA(bytes.fromhex(vaa), client, player3, self.tokenid)


        print("testid balance after = ", self.getBalances(client, get_application_address(self.testid)))

#        sys.exit(0)

        print(".. Ok, now it is time to up the message fees")

        bal = self.getBalances(client, get_application_address(self.coreid))
        print("core contract has " + str(bal) + " algo (" + get_application_address(self.coreid) + ")")
        print("core contract has a MessageFee set to " + str(self.getMessageFee()))

        seq += 1
        v = gt.genGSetFee(gt.guardianPrivKeys, 2, seq, seq, 2000000)
        self.submitVAA(bytes.fromhex(v), client, player, self.coreid)
        seq += 1

        print("core contract now has a MessageFee set to " + str(self.getMessageFee()))

#        v = gt.genGSetFee(gt.guardianPrivKeys, 2, seq, seq, 0)
#        self.submitVAA(bytes.fromhex(v), client, player, self.coreid)
#        seq += 1

#        print("core contract is back to  " + str(self.getMessageFee()))

        print("Generating an attest.. This will cause a message to get published .. which should cause fees to get sent to the core contract")
        sid = self.testAttest(client, player2, self.testasset)
        print("... track down the generated VAA")
        vaa = self.getVAA(client, player, sid, self.tokenid)
        v = self.parseVAA(bytes.fromhex(vaa))
        print("We got a " + v["Meta"])

        bal = self.getBalances(client, get_application_address(self.coreid))
        print("core contract has " + str(bal) + " algo (" + get_application_address(self.coreid) + ")")

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

if __name__ == "__main__":
    core = AlgoTest()
    core.simple_test()
