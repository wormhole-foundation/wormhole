# python3 -m pip install pycryptodomex uvarint pyteal web3 coincurve

from time import time, sleep
from eth_abi import encode_single, encode_abi
from typing import List, Tuple, Dict, Any, Optional, Union
from base64 import b64decode
import base64
import random
import time
import hashlib
import uuid
import sys
import json
import uvarint
from local_blob import LocalBlob
from portal_core import getCoreContracts
from TmplSig import TmplSig
import argparse
from gentest import GenTest

from algosdk.v2client.algod import AlgodClient
from algosdk.kmd import KMDClient
from algosdk import account, mnemonic
from algosdk.encoding import decode_address, encode_address
from algosdk.future import transaction
from pyteal import compileTeal, Mode, Expr
from pyteal import *
from algosdk.logic import get_application_address
from vaa_verify import get_vaa_verify

from Cryptodome.Hash import keccak

from algosdk.future.transaction import LogicSig

from token_bridge import get_token_bridge

from algosdk.v2client import indexer

import pprint

max_keys = 16
max_bytes_per_key = 127
bits_per_byte = 8

bits_per_key = max_bytes_per_key * bits_per_byte
max_bytes = max_bytes_per_key * max_keys
max_bits = bits_per_byte * max_bytes

class Account:
    """Represents a private key and address for an Algorand account"""

    def __init__(self, privateKey: str) -> None:
        self.sk = privateKey
        self.addr = account.address_from_private_key(privateKey)
        #print (privateKey + " -> " + self.getMnemonic())

    def getAddress(self) -> str:
        return self.addr

    def getPrivateKey(self) -> str:
        return self.sk

    def getMnemonic(self) -> str:
        return mnemonic.from_private_key(self.sk)

    @classmethod
    def FromMnemonic(cls, m: str) -> "Account":
        return cls(mnemonic.to_private_key(m))

class PendingTxnResponse:
    def __init__(self, response: Dict[str, Any]) -> None:
        self.poolError: str = response["pool-error"]
        self.txn: Dict[str, Any] = response["txn"]

        self.applicationIndex: Optional[int] = response.get("application-index")
        self.assetIndex: Optional[int] = response.get("asset-index")
        self.closeRewards: Optional[int] = response.get("close-rewards")
        self.closingAmount: Optional[int] = response.get("closing-amount")
        self.confirmedRound: Optional[int] = response.get("confirmed-round")
        self.globalStateDelta: Optional[Any] = response.get("global-state-delta")
        self.localStateDelta: Optional[Any] = response.get("local-state-delta")
        self.receiverRewards: Optional[int] = response.get("receiver-rewards")
        self.senderRewards: Optional[int] = response.get("sender-rewards")

        self.innerTxns: List[Any] = response.get("inner-txns", [])
        self.logs: List[bytes] = [b64decode(l) for l in response.get("logs", [])]

class PortalCore:
    def __init__(self) -> None:
        self.gt = None
        self.foundation = None

        self.ALGOD_ADDRESS = "http://localhost:4001"
        self.ALGOD_TOKEN = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        self.FUNDING_AMOUNT = 100_000_000_000

        self.KMD_ADDRESS = "http://localhost:4002"
        self.KMD_TOKEN = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        self.KMD_WALLET_NAME = "unencrypted-default-wallet"
        self.KMD_WALLET_PASSWORD = ""

        self.INDEXER_TOKEN = "a" * 64
        self.INDEXER_ADDRESS = 'http://localhost:8980'
        self.INDEXER_ROUND = 0
        self.NOTE_PREFIX = 'publishMessage'.encode()

        self.myindexer = indexer.IndexerClient(indexer_token=self.INDEXER_TOKEN, indexer_address=self.INDEXER_ADDRESS)

        self.seed_amt = int(1002000)  # The black magic in this number... 
        self.cache = {}
        self.asset_cache = {}

        self.kmdAccounts : Optional[List[Account]] = None

        self.accountList : List[Account] = []
        self.zeroPadBytes = "00"*32

        self.tsig = TmplSig("sig")

        self.client = self.getAlgodClient()
        self.vaa_verify = self.client.compile(get_vaa_verify())
        self.vaa_verify["lsig"] = LogicSig(base64.b64decode(self.vaa_verify["result"]))

    def init(self, args) -> None:
        self.args = args
        self.ALGOD_ADDRESS = args.algod_address
        self.ALGOD_TOKEN = args.algod_token
        self.KMD_ADDRESS = args.kmd_address
        self.KMD_TOKEN = args.kmd_token
        self.KMD_WALLET_NAME = args.kmd_name
        self.KMD_WALLET_PASSWORD = args.kmd_password
        self.TARGET_ACCOUNT = args.mnemonic

    def waitForTransaction(
            self, client: AlgodClient, txID: str, timeout: int = 10
    ) -> PendingTxnResponse:
        lastStatus = client.status()
        lastRound = lastStatus["last-round"]
        startRound = lastRound
    
        while lastRound < startRound + timeout:
            pending_txn = client.pending_transaction_info(txID)
    
            if pending_txn.get("confirmed-round", 0) > 0:
                return PendingTxnResponse(pending_txn)
    
            if pending_txn["pool-error"]:
                raise Exception("Pool error: {}".format(pending_txn["pool-error"]))
    
            lastStatus = client.status_after_block(lastRound + 1)
    
            lastRound += 1
    
        raise Exception(
            "Transaction {} not confirmed after {} rounds".format(txID, timeout)
        )

    def getKmdClient(self) -> KMDClient:
        return KMDClient(self.KMD_TOKEN, self.KMD_ADDRESS)
    
    def getGenesisAccounts(self) -> List[Account]:
        if self.kmdAccounts is None:
            kmd = self.getKmdClient()
    
            wallets = kmd.list_wallets()
            walletID = None
            for wallet in wallets:
                if wallet["name"] == self.KMD_WALLET_NAME:
                    walletID = wallet["id"]
                    break
    
            if walletID is None:
                raise Exception("Wallet not found: {}".format(self.KMD_WALLET_NAME))
    
            walletHandle = kmd.init_wallet_handle(walletID, self.KMD_WALLET_PASSWORD)
    
            try:
                addresses = kmd.list_keys(walletHandle)
                privateKeys = [
                    kmd.export_key(walletHandle, self.KMD_WALLET_PASSWORD, addr)
                    for addr in addresses
                ]
                self.kmdAccounts = [Account(sk) for sk in privateKeys]
            finally:
                kmd.release_wallet_handle(walletHandle)
    
        return self.kmdAccounts
    
    def getTemporaryAccount(self, client: AlgodClient) -> Account:
        if len(self.accountList) == 0:
            sks = [account.generate_account()[0] for i in range(3)]
            self.accountList = [Account(sk) for sk in sks]
    
            genesisAccounts = self.getGenesisAccounts()
            suggestedParams = client.suggested_params()
    
            txns: List[transaction.Transaction] = []
            for i, a in enumerate(self.accountList):
                fundingAccount = genesisAccounts[i % len(genesisAccounts)]
                txns.append(
                    transaction.PaymentTxn(
                        sender=fundingAccount.getAddress(),
                        receiver=a.getAddress(),
                        amt=self.FUNDING_AMOUNT,
                        sp=suggestedParams,
                    )
                )
    
            txns = transaction.assign_group_id(txns)
            signedTxns = [
                txn.sign(genesisAccounts[i % len(genesisAccounts)].getPrivateKey())
                for i, txn in enumerate(txns)
            ]
    
            client.send_transactions(signedTxns)
    
            self.waitForTransaction(client, signedTxns[0].get_txid())
    
        return self.accountList.pop()
    
    def getAlgodClient(self) -> AlgodClient:
        return AlgodClient(self.ALGOD_TOKEN, self.ALGOD_ADDRESS)

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

    def fullyCompileContract(self, client: AlgodClient, contract: Expr) -> bytes:
        teal = compileTeal(contract, mode=Mode.Application, version=6)
        response = client.compile(teal)
        return response

    # helper function that formats global state for printing
    def format_state(self, state):
        formatted = {}
        for item in state:
            key = item['key']
            value = item['value']
            formatted_key = base64.b64decode(key).decode('utf-8')
            if value['type'] == 1:
                # byte string
                if formatted_key == 'voted':
                    formatted_value = base64.b64decode(value['bytes']).decode('utf-8')
                else:
                    formatted_value = value['bytes']
                formatted[formatted_key] = formatted_value
            else:
                # integer
                formatted[formatted_key] = value['uint']
        return formatted
    
    # helper function to read app global state
    def read_global_state(self, client, addr, app_id):
        results = client.account_info(addr)
        apps_created = results['created-apps']
        for app in apps_created:
            if app['id'] == app_id and 'global-state' in app['params']:
                return self.format_state(app['params']['global-state'])
        return {}

    def read_state(self, client, addr, app_id):
        results = client.account_info(addr)
        apps_created = results['created-apps']
        for app in apps_created:
            if app['id'] == app_id:
                return app
        return {}

    def encoder(self, type, val):
        if type == 'uint8':
            return encode_single(type, val).hex()[62:64]
        if type == 'uint16':
            return encode_single(type, val).hex()[60:64]
        if type == 'uint32':
            return encode_single(type, val).hex()[56:64]
        if type == 'uint64':
            return encode_single(type, val).hex()[64-(16):64]
        if type == 'uint128':
            return encode_single(type, val).hex()[64-(32):64]
        if type == 'uint256' or type == 'bytes32':
            return encode_single(type, val).hex()[64-(64):64]
        raise Exception("you suck")

    def devnetUpgradeVAA(self):
        v = self.genUpgradePayload()
        print("The payload: " + str(v))
        if self.gt == None:
            self.gt = GenTest(False)

        emitter = bytes.fromhex(self.zeroPadBytes[0:(31*2)] + "04")

        guardianSet = self.getGovSet()
        nonce = int(random.random() * 20000)
        ret = [
            self.gt.createSignedVAA(guardianSet, self.gt.guardianPrivKeys, int(time.time()), nonce, 1, emitter, int(random.random() * 20000), 32, 8, v[0]),
            self.gt.createSignedVAA(guardianSet, self.gt.guardianPrivKeys, int(time.time()), nonce, 1, emitter, int(random.random() * 20000), 32, 8, v[1]),
        ]
        
#        pprint.pprint(self.parseVAA(bytes.fromhex(ret[0])))
#        pprint.pprint(self.parseVAA(bytes.fromhex(ret[1])))

        return ret

    def getGovSet(self):
        s = self.client.application_info(self.coreid)["params"]["global-state"]
        k = base64.b64encode(b"currentGuardianSetIndex").decode('utf-8')
        for x in s:
            if x["key"] == k:
                return x["value"]["uint"]
        return -1

    def genUpgradePayload(self):
        approval, clear = getCoreContracts(self.client, seed_amt=self.seed_amt, tmpl_sig=self.tsig)

        b  = self.zeroPadBytes[0:(28*2)]
        b += self.encoder("uint8", ord("C"))
        b += self.encoder("uint8", ord("o"))
        b += self.encoder("uint8", ord("r"))
        b += self.encoder("uint8", ord("e"))
        b += self.encoder("uint8", 1)
        b += self.encoder("uint16", 8)

        b += decode_address(approval["hash"]).hex()

        ret = [b]

        approval, clear = get_token_bridge(self.client, seed_amt=self.seed_amt, tmpl_sig=self.tsig)

        b  = self.zeroPadBytes[0:((32 -11)*2)]
        b += self.encoder("uint8", ord("T"))
        b += self.encoder("uint8", ord("o"))
        b += self.encoder("uint8", ord("k"))
        b += self.encoder("uint8", ord("e"))
        b += self.encoder("uint8", ord("n"))
        b += self.encoder("uint8", ord("B"))
        b += self.encoder("uint8", ord("r"))
        b += self.encoder("uint8", ord("i"))
        b += self.encoder("uint8", ord("d"))
        b += self.encoder("uint8", ord("g"))
        b += self.encoder("uint8", ord("e"))

        b += self.encoder("uint8", 2)  # action
        b += self.encoder("uint16", 8) # target chain
        b += decode_address(approval["hash"]).hex()
        print(decode_address(approval["hash"]).hex())

        ret.append(b)
        return ret

    def createPortalCoreApp(
        self,
        client: AlgodClient,
        sender: Account,
    ) -> int:
        approval, clear = getCoreContracts(client, seed_amt=self.seed_amt, tmpl_sig=self.tsig)

        globalSchema = transaction.StateSchema(num_uints=8, num_byte_slices=16)
        localSchema = transaction.StateSchema(num_uints=0, num_byte_slices=16)
    
        app_args = [ ]
    
        txn = transaction.ApplicationCreateTxn(
            sender=sender.getAddress(),
            on_complete=transaction.OnComplete.NoOpOC,
            approval_program=b64decode(approval["result"]),
            clear_program=b64decode(clear["result"]),
            global_schema=globalSchema,
            local_schema=localSchema,
            extra_pages = 1,
            app_args=app_args,
            sp=client.suggested_params(),
        )
    
        signedTxn = txn.sign(sender.getPrivateKey())
    
        client.send_transaction(signedTxn)
    
        response = self.waitForTransaction(client, signedTxn.get_txid())
        assert response.applicationIndex is not None and response.applicationIndex > 0

        # Lets give it a bit of money so that it is not a "ghost" account
        txn = transaction.PaymentTxn(sender = sender.getAddress(), sp = client.suggested_params(), receiver = get_application_address(response.applicationIndex), amt = 100000)
        signedTxn = txn.sign(sender.getPrivateKey())
        client.send_transaction(signedTxn)

        return response.applicationIndex

    def createTokenBridgeApp(
        self,
        client: AlgodClient,
        sender: Account,
    ) -> int:
        approval, clear = get_token_bridge(client, seed_amt=self.seed_amt, tmpl_sig=self.tsig)

        if len(b64decode(approval["result"])) > 4060:
            print("token bridge contract is too large... This might prevent updates later")

        globalSchema = transaction.StateSchema(num_uints=4, num_byte_slices=30)
        localSchema = transaction.StateSchema(num_uints=0, num_byte_slices=16)
    
        app_args = [self.coreid]

        txn = transaction.ApplicationCreateTxn(
            sender=sender.getAddress(),
            on_complete=transaction.OnComplete.NoOpOC,
            approval_program=b64decode(approval["result"]),
            clear_program=b64decode(clear["result"]),
            global_schema=globalSchema,
            local_schema=localSchema,
            app_args=app_args,
            extra_pages = 2,
            sp=client.suggested_params(),
        )
    
        signedTxn = txn.sign(sender.getPrivateKey())
    
        client.send_transaction(signedTxn)
    
        response = self.waitForTransaction(client, signedTxn.get_txid())
        assert response.applicationIndex is not None and response.applicationIndex > 0

        # Lets give it a bit of money so that it is not a "ghost" account
        txn = transaction.PaymentTxn(sender = sender.getAddress(), sp = client.suggested_params(), receiver = get_application_address(response.applicationIndex), amt = 100000)
        signedTxn = txn.sign(sender.getPrivateKey())
        client.send_transaction(signedTxn)

        return response.applicationIndex

    def account_exists(self, client, app_id, addr):
        try:
            ai = client.account_info(addr)
            if "apps-local-state" not in ai:
                return False
    
            for app in ai["apps-local-state"]:
                if app["id"] == app_id:
                    return True
        except:
            print("Failed to find account {}".format(addr))
        return False

    def optin(self, client, sender, app_id, idx, emitter, doCreate=True):
        aa = decode_address(get_application_address(app_id)).hex()

        lsa = self.tsig.populate(
            {
                "TMPL_SEED_AMT": self.seed_amt,
                "TMPL_APP_ID": app_id,
                "TMPL_APP_ADDRESS": aa,
                "TMPL_ADDR_IDX": idx,
                "TMPL_EMITTER_ID": emitter,
            }
        )

        sig_addr = lsa.address()

        if sig_addr not in self.cache and not self.account_exists(client, app_id, sig_addr):
            if doCreate:
#                pprint.pprint(("Creating", app_id, idx, emitter, sig_addr))

                # Create it
                sp = client.suggested_params()
    
                seed_txn = transaction.PaymentTxn(sender = sender.getAddress(), 
                                                  sp = sp, 
                                                  receiver = sig_addr, 
                                                  amt = self.seed_amt)
                optin_txn = transaction.ApplicationOptInTxn(sig_addr, sp, app_id)
                rekey_txn = transaction.PaymentTxn(sender=sig_addr, sp=sp, receiver=sig_addr, 
                                                   amt=0, rekey_to=get_application_address(app_id))
    
                transaction.assign_group_id([seed_txn, optin_txn, rekey_txn])
    
                signed_seed = seed_txn.sign(sender.getPrivateKey())
                signed_optin = transaction.LogicSigTransaction(optin_txn, lsa)
                signed_rekey = transaction.LogicSigTransaction(rekey_txn, lsa)
    
                client.send_transactions([signed_seed, signed_optin, signed_rekey])
                self.waitForTransaction(client, signed_optin.get_txid())
                
                self.cache[sig_addr] = True

        return sig_addr

    def parseSeqFromLog(self, txn):
        return int.from_bytes(b64decode(txn.innerTxns[0]["logs"][0]), "big")

    def getCreator(self, client, sender, asset_id):
        return client.asset_info(asset_id)["params"]["creator"]

    def sendTxn(self, client, sender, txns, doWait):
        transaction.assign_group_id(txns)

        grp = []
        pk = sender.getPrivateKey()
        for t in txns:
            grp.append(t.sign(pk))

        client.send_transactions(grp)
        if doWait:
            return self.waitForTransaction(client, grp[-1].get_txid())
        else:
            return grp[-1].get_txid()

    def bootGuardians(self, vaa, client, sender, coreid):
        p = self.parseVAA(vaa)
        if "NewGuardianSetIndex" not in p:
            raise Exception("invalid guardian VAA")

        seq_addr = self.optin(client, sender, coreid, int(p["sequence"] / max_bits), p["chainRaw"].hex() + p["emitter"].hex())
        guardian_addr = self.optin(client, sender, coreid, p["index"], b"guardian".hex())
        newguardian_addr = self.optin(client, sender, coreid, p["NewGuardianSetIndex"], b"guardian".hex())

        # wormhole is not a cheap protocol... we need to buy ourselves
        # some extra CPU cycles by having an early txn do nothing.
        # This leaves cycles over for later txn's in the same group

        txn0 = transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=coreid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"nop", b"0"],
            sp=client.suggested_params(),
        )

        txn1 = transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=coreid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"nop", b"1"],
            sp=client.suggested_params(),
        )

        txn2 = transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=coreid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"init", vaa, decode_address(self.vaa_verify["hash"])],
            accounts=[seq_addr, guardian_addr, newguardian_addr],
            sp=client.suggested_params(),
        )

        transaction.assign_group_id([txn0, txn1, txn2])
    
        signedTxn0 = txn0.sign(sender.getPrivateKey())
        signedTxn1 = txn1.sign(sender.getPrivateKey())
        signedTxn2 = txn2.sign(sender.getPrivateKey())

        client.send_transactions([signedTxn0, signedTxn1, signedTxn2])
        response = self.waitForTransaction(client, signedTxn2.get_txid())
        #pprint.pprint(response.__dict__)

    def decodeLocalState(self, client, sender, appid, addr):
        app_state = None
        ai = client.account_info(addr)
        for app in ai["apps-local-state"]:
            if app["id"] == appid:
                app_state = app["key-value"]

        ret = b''
        if None != app_state:
            vals = {}
            e = bytes.fromhex("00"*127)
            for kv in app_state:
                key = int.from_bytes(base64.b64decode(kv["key"]), "big")
                v = base64.b64decode(kv["value"]["bytes"])
                if v != e:
                    vals[key] = v
            for k in sorted(vals.keys()):
                ret = ret + vals[k]
        return ret

    # There is no client side duplicate suppression, error checking, or validity
    # checking. We need to be able to detect all failure cases in
    # the contract itself and we want to use this to drive the failure test
    # cases

    def simpleVAA(self, vaa, client, sender, appid):
        p = {"version": int.from_bytes(vaa[0:1], "big"), "index": int.from_bytes(vaa[1:5], "big"), "siglen": int.from_bytes(vaa[5:6], "big")}
        ret["signatures"] = vaa[6:(ret["siglen"] * 66) + 6]
        ret["sigs"] = []
        for i in range(ret["siglen"]):
            ret["sigs"].append(vaa[(6 + (i * 66)):(6 + (i * 66)) + 66].hex())
        off = (ret["siglen"] * 66) + 6
        ret["digest"] = vaa[off:]  # This is what is actually signed...
        ret["timestamp"] = int.from_bytes(vaa[off:(off + 4)], "big")
        off += 4
        ret["nonce"] = int.from_bytes(vaa[off:(off + 4)], "big")
        off += 4
        ret["chainRaw"] = vaa[off:(off + 2)]
        ret["chain"] = int.from_bytes(vaa[off:(off + 2)], "big")
        off += 2
        ret["emitter"] = vaa[off:(off + 32)]
        off += 32
        ret["sequence"] = int.from_bytes(vaa[off:(off + 8)], "big")
        off += 8
        ret["consistency"] = int.from_bytes(vaa[off:(off + 1)], "big")
        off += 1

        seq_addr = self.optin(client, sender, appid, int(p["sequence"] / max_bits), p["chainRaw"].hex() + p["emitter"].hex())
        # And then the signatures to help us verify the vaa_s
        guardian_addr = self.optin(client, sender, self.coreid, p["index"], b"guardian".hex())

        accts = [seq_addr, guardian_addr]

        keys = self.decodeLocalState(client, sender, self.coreid, guardian_addr)

        sp = client.suggested_params()

        txns = []

        # Right now there is not really a good way to estimate the fees,
        # in production, on a conjested network, how much verifying
        # the signatures is going to cost.

        # So, what we do instead
        # is we top off the verifier back up to 2A so effectively we
        # are paying for the previous persons overrage which on a
        # unconjested network should be zero

        pmt = 3000
        bal = self.getBalances(client, self.vaa_verify["hash"])
        if ((200000 - bal[0]) >= pmt):
            pmt = 200000 - bal[0]

        #print("Sending %d algo to cover fees" % (pmt))
        txns.append(
            transaction.PaymentTxn(
                sender = sender.getAddress(), 
                sp = sp, 
                receiver = self.vaa_verify["hash"], 
                amt = pmt * 2
            )
        )

        # How many signatures can we process in a single txn... we can do 9!
        bsize = (9*66)
        blocks = int(len(p["signatures"]) / bsize) + 1

        # We don't pass the entire payload in but instead just pass it pre digested.  This gets around size
        # limitations with lsigs AND reduces the cost of the entire operation on a conjested network by reducing the
        # bytes passed into the transaction
        digest = keccak.new(digest_bits=256).update(keccak.new(digest_bits=256).update(p["digest"]).digest()).digest()

        for i in range(blocks):
            # Which signatures will we be verifying in this block
            sigs = p["signatures"][(i * bsize):]
            if (len(sigs) > bsize):
                sigs = sigs[:bsize]
            # keys
            kset = b''
            # Grab the key associated the signature
            for q in range(int(len(sigs) / 66)):
                # Which guardian is this signature associated with
                g = sigs[q * 66]
                key = keys[((g * 20) + 1) : (((g + 1) * 20) + 1)]
                kset = kset + key

            txns.append(transaction.ApplicationCallTxn(
                    sender=self.vaa_verify["hash"],
                    index=self.coreid,
                    on_complete=transaction.OnComplete.NoOpOC,
                    app_args=[b"verifySigs", sigs, kset, digest],
                    accounts=accts,
                    sp=sp
                ))

        txns.append(transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=self.coreid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"verifyVAA", vaa],
            accounts=accts,
            sp=sp
        ))

        return txns

    def signVAA(self, client, sender, txns):
        transaction.assign_group_id(txns)

        grp = []
        pk = sender.getPrivateKey()
        for t in txns:
            if ("app_args" in t.__dict__ and len(t.app_args) > 0 and t.app_args[0] == b"verifySigs"):
                grp.append(transaction.LogicSigTransaction(t, self.vaa_verify["lsig"]))
            else:
                grp.append(t.sign(pk))

        client.send_transactions(grp)
        ret = []
        for x in grp:
            response = self.waitForTransaction(client, x.get_txid())
            if "logs" in response.__dict__ and len(response.__dict__["logs"]) > 0:
                ret.append(response.__dict__["logs"])
        return ret

    def submitVAA(self, vaa, client, sender, appid):
        # A lot of our logic here depends on parseVAA and knowing what the payload is..
        p = self.parseVAA(vaa)

        #pprint.pprint(p)

        seq_addr = self.optin(client, sender, appid, int(p["sequence"] / max_bits), p["chainRaw"].hex() + p["emitter"].hex())
        # And then the signatures to help us verify the vaa_s
        guardian_addr = self.optin(client, sender, self.coreid, p["index"], b"guardian".hex())

        accts = [seq_addr, guardian_addr]

        # If this happens to be setting up a new guardian set, we probably need it as well...
        if p["Meta"] == "CoreGovernance" and p["action"] == 2:
            newguardian_addr = self.optin(client, sender, self.coreid, p["NewGuardianSetIndex"], b"guardian".hex())
            accts.append(newguardian_addr)

        # When we attest for a new token, we need some place to store the info... later we will need to 
        # mirror the other way as well
        if p["Meta"] == "TokenBridge Attest" or p["Meta"] == "TokenBridge Transfer" or p["Meta"] == "TokenBridge Transfer With Payload":
            if p["FromChain"] != 8:
                chain_addr = self.optin(client, sender, self.tokenid, p["FromChain"], p["Contract"])
            else:
                asset_id = int.from_bytes(bytes.fromhex(p["Contract"]), "big")
                chain_addr = self.optin(client, sender, self.tokenid, asset_id, b"native".hex())
            accts.append(chain_addr)

        keys = self.decodeLocalState(client, sender, self.coreid, guardian_addr)

        sp = client.suggested_params()

        txns = []

        # Right now there is not really a good way to estimate the fees,
        # in production, on a conjested network, how much verifying
        # the signatures is going to cost.

        # So, what we do instead
        # is we top off the verifier back up to 2A so effectively we
        # are paying for the previous persons overrage which on a
        # unconjested network should be zero

        pmt = 3000
        bal = self.getBalances(client, self.vaa_verify["hash"])
        if ((200000 - bal[0]) >= pmt):
            pmt = 200000 - bal[0]

        #print("Sending %d algo to cover fees" % (pmt))
        txns.append(
            transaction.PaymentTxn(
                sender = sender.getAddress(), 
                sp = sp, 
                receiver = self.vaa_verify["hash"], 
                amt = pmt * 2
            )
        )

        # How many signatures can we process in a single txn... we can do 9!
        bsize = (9*66)
        blocks = int(len(p["signatures"]) / bsize) + 1

        # We don't pass the entire payload in but instead just pass it pre digested.  This gets around size
        # limitations with lsigs AND reduces the cost of the entire operation on a conjested network by reducing the
        # bytes passed into the transaction
        digest = keccak.new(digest_bits=256).update(keccak.new(digest_bits=256).update(p["digest"]).digest()).digest()

        for i in range(blocks):
            # Which signatures will we be verifying in this block
            sigs = p["signatures"][(i * bsize):]
            if (len(sigs) > bsize):
                sigs = sigs[:bsize]
            # keys
            kset = b''
            # Grab the key associated the signature
            for q in range(int(len(sigs) / 66)):
                # Which guardian is this signature associated with
                g = sigs[q * 66]
                key = keys[((g * 20) + 1) : (((g + 1) * 20) + 1)]
                kset = kset + key

            txns.append(transaction.ApplicationCallTxn(
                    sender=self.vaa_verify["hash"],
                    index=self.coreid,
                    on_complete=transaction.OnComplete.NoOpOC,
                    app_args=[b"verifySigs", sigs, kset, digest],
                    accounts=accts,
                    sp=sp
                ))

        txns.append(transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=self.coreid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"verifyVAA", vaa],
            accounts=accts,
            sp=sp
        ))

        if p["Meta"] == "CoreGovernance":
            txns.append(transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=self.coreid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"governance", vaa],
                accounts=accts,
                note = p["digest"],
                sp=sp
            ))

        if p["Meta"] == "TokenBridge RegisterChain" or p["Meta"] == "TokenBridge UpgradeContract":
            txns.append(transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=self.tokenid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"governance", vaa],
                accounts=accts,
                note = p["digest"],
                foreign_apps = [self.coreid],
                sp=sp
            ))

        if p["Meta"] == "TokenBridge Attest":
            # if we DO decode it, we can do a sanity check... of
            # course, the hacker might NOT decode it so we have to
            # handle both cases...

            asset = (self.decodeLocalState(client, sender, self.tokenid, chain_addr))
            foreign_assets = []
            if (len(asset) > 8):
                foreign_assets.append(int.from_bytes(asset[0:8], "big"))

            txns.append(
                transaction.PaymentTxn(
                    sender = sender.getAddress(),
                    sp = sp, 
                    receiver = chain_addr,
                    amt = 100000
                )
            )

            txns.append(transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=self.tokenid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"nop", 1],
                sp=sp
            ))

            txns.append(transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=self.tokenid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"nop", 2],
                sp=sp
            ))

            txns.append(transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=self.tokenid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"receiveAttest", vaa],
                accounts=accts,
                foreign_assets = foreign_assets,
                sp=sp
            ))
            txns[-1].fee = txns[-1].fee * 2

        if p["Meta"] == "TokenBridge Transfer" or p["Meta"] == "TokenBridge Transfer With Payload":
            foreign_assets = []
            a = 0
            if p["FromChain"] != 8:
                asset = (self.decodeLocalState(client, sender, self.tokenid, chain_addr))
                if (len(asset) > 8):
                    a = int.from_bytes(asset[0:8], "big")
            else:
                a = int.from_bytes(bytes.fromhex(p["Contract"]), "big")

            # The receiver needs to be optin in to receive the coins... Yeah, the relayer pays for this

            addr = encode_address(bytes.fromhex(p["ToAddress"]))

            if a != 0:
                foreign_assets.append(a)
                self.asset_optin(client, sender, foreign_assets[0], addr)
                # And this is how the relayer gets paid...
                if p["Fee"] != self.zeroPadBytes:
                    self.asset_optin(client, sender, foreign_assets[0], sender.getAddress())

            accts.append(addr)

            txns.append(transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=self.tokenid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"receiveTransfer", vaa],
                accounts=accts,
                foreign_assets = foreign_assets,
                sp=sp
            ))

            # We need to cover the inner transactions
            if p["Fee"] != self.zeroPadBytes:
                txns[-1].fee = txns[-1].fee * 3
            else:
                txns[-1].fee = txns[-1].fee * 2

        transaction.assign_group_id(txns)

        grp = []
        pk = sender.getPrivateKey()
        for t in txns:
            if ("app_args" in t.__dict__ and len(t.app_args) > 0 and t.app_args[0] == b"verifySigs"):
                grp.append(transaction.LogicSigTransaction(t, self.vaa_verify["lsig"]))
            else:
                grp.append(t.sign(pk))

        client.send_transactions(grp)
        ret = []
        for x in grp:
            response = self.waitForTransaction(client, x.get_txid())
            if "logs" in response.__dict__ and len(response.__dict__["logs"]) > 0:
                ret.append(response.__dict__["logs"])
        return ret

    def parseVAA(self, vaa):
#        print (vaa.hex())
        ret = {"version": int.from_bytes(vaa[0:1], "big"), "index": int.from_bytes(vaa[1:5], "big"), "siglen": int.from_bytes(vaa[5:6], "big")}
        ret["signatures"] = vaa[6:(ret["siglen"] * 66) + 6]
        ret["sigs"] = []
        for i in range(ret["siglen"]):
            ret["sigs"].append(vaa[(6 + (i * 66)):(6 + (i * 66)) + 66].hex())
        off = (ret["siglen"] * 66) + 6
        ret["digest"] = vaa[off:]  # This is what is actually signed...
        ret["timestamp"] = int.from_bytes(vaa[off:(off + 4)], "big")
        off += 4
        ret["nonce"] = int.from_bytes(vaa[off:(off + 4)], "big")
        off += 4
        ret["chainRaw"] = vaa[off:(off + 2)]
        ret["chain"] = int.from_bytes(vaa[off:(off + 2)], "big")
        off += 2
        ret["emitter"] = vaa[off:(off + 32)]
        off += 32
        ret["sequence"] = int.from_bytes(vaa[off:(off + 8)], "big")
        off += 8
        ret["consistency"] = int.from_bytes(vaa[off:(off + 1)], "big")
        off += 1

        ret["Meta"] = "Unknown"

        if vaa[off:(off + 32)].hex() == "000000000000000000000000000000000000000000546f6b656e427269646765":
            ret["Meta"] = "TokenBridge"
            ret["module"] = vaa[off:(off + 32)].hex()
            off += 32
            ret["action"] = int.from_bytes(vaa[off:(off + 1)], "big")
            off += 1
            if ret["action"] == 1:
                ret["Meta"] = "TokenBridge RegisterChain"
                ret["targetChain"] = int.from_bytes(vaa[off:(off + 2)], "big")
                off += 2
                ret["EmitterChainID"] = int.from_bytes(vaa[off:(off + 2)], "big")
                off += 2
                ret["targetEmitter"] = vaa[off:(off + 32)].hex()
                off += 32
            if ret["action"] == 2:
                ret["Meta"] = "TokenBridge UpgradeContract"
                ret["targetChain"] = int.from_bytes(vaa[off:(off + 2)], "big")
                off += 2
                ret["newContract"] = vaa[off:(off + 32)].hex()
                off += 32

        if vaa[off:(off + 32)].hex() == "00000000000000000000000000000000000000000000000000000000436f7265":
            ret["Meta"] = "CoreGovernance"
            ret["module"] = vaa[off:(off + 32)].hex()
            off += 32
            ret["action"] = int.from_bytes(vaa[off:(off + 1)], "big")
            off += 1
            ret["targetChain"] = int.from_bytes(vaa[off:(off + 2)], "big")
            off += 2
            if ret["action"] == 2:
                ret["NewGuardianSetIndex"] = int.from_bytes(vaa[off:(off + 4)], "big")
            else:
                ret["Contract"] = vaa[off:(off + 32)].hex()

        if ((len(vaa[off:])) == 100) and int.from_bytes((vaa[off:off+1]), "big") == 2:
            ret["Meta"] = "TokenBridge Attest"
            ret["Type"] = int.from_bytes((vaa[off:off+1]), "big")
            off += 1
            ret["Contract"] = vaa[off:(off + 32)].hex()
            off += 32
            ret["FromChain"] = int.from_bytes(vaa[off:(off + 2)], "big")
            off += 2
            ret["Decimals"] = int.from_bytes((vaa[off:off+1]), "big")
            off += 1
            ret["Symbol"] = vaa[off:(off + 32)].hex()
            off += 32
            ret["Name"] = vaa[off:(off + 32)].hex()

        if ((len(vaa[off:])) == 133) and int.from_bytes((vaa[off:off+1]), "big") == 1:
            ret["Meta"] = "TokenBridge Transfer"
            ret["Type"] = int.from_bytes((vaa[off:off+1]), "big")
            off += 1
            ret["Amount"] = vaa[off:(off + 32)].hex()
            off += 32
            ret["Contract"] = vaa[off:(off + 32)].hex()
            off += 32
            ret["FromChain"] = int.from_bytes(vaa[off:(off + 2)], "big")
            off += 2
            ret["ToAddress"] = vaa[off:(off + 32)].hex()
            off += 32
            ret["ToChain"] = int.from_bytes(vaa[off:(off + 2)], "big")
            off += 2
            ret["Fee"] = vaa[off:(off + 32)].hex()

        if int.from_bytes((vaa[off:off+1]), "big") == 3:
            ret["Meta"] = "TokenBridge Transfer With Payload"
            ret["Type"] = int.from_bytes((vaa[off:off+1]), "big")
            off += 1
            ret["Amount"] = vaa[off:(off + 32)].hex()
            off += 32
            ret["Contract"] = vaa[off:(off + 32)].hex()
            off += 32
            ret["FromChain"] = int.from_bytes(vaa[off:(off + 2)], "big")
            off += 2
            ret["ToAddress"] = vaa[off:(off + 32)].hex()
            off += 32
            ret["ToChain"] = int.from_bytes(vaa[off:(off + 2)], "big")
            off += 2
            ret["Fee"] = vaa[off:(off + 32)].hex()
            off += 32
            ret["Payload"] = vaa[off:].hex()
        
        return ret

    def dev_deploy(self):
        print("")

        print("Creating the PortalCore app")
        self.coreid = self.createPortalCoreApp(client=self.client, sender=self.foundation)
        print("coreid = " + str(self.coreid))

        print("Create the token bridge")
        self.tokenid = self.createTokenBridgeApp(self.client, self.foundation)
        print("token bridge " + str(self.tokenid) + " address " + get_application_address(self.tokenid))

        print("bootstrapping the guardian set...")
        bootVAA = bytes.fromhex("0100000001010001ca2fbf60ac6227d47dda4fe2e7bccc087f27d22170a212b9800da5b4cbf0d64c52deb2f65ce58be2267bf5b366437c267b5c7b795cd6cea1ac2fee8a1db3ad006225f801000000010001000000000000000000000000000000000000000000000000000000000000000400000000000000012000000000000000000000000000000000000000000000000000000000436f72650200000000000001beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe")
        self.bootGuardians(bootVAA, self.client, self.foundation, self.coreid)

        print("bootstrapping the chain registrationst...")

        vaas = [
            # Solana
            "01000000000100c9f4230109e378f7efc0605fb40f0e1869f2d82fda5b1dfad8a5a2dafee85e033d155c18641165a77a2db6a7afbf2745b458616cb59347e89ae0c7aa3e7cc2d400000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e4272696467650100000001c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f",
        ]

        print("Registering chains")

        for v in vaas:
            print("Submitting: " + v)
            self.submitVAA(bytes.fromhex(v), self.client, self.foundation, self.tokenid)


    def updateCore(self) -> None:
        print("Updating the core contracts")
        approval, clear = getCoreContracts(self.client, seed_amt=self.seed_amt, tmpl_sig=self.tsig)

#        print(decode_address(clear["hash"]).hex())
#        sys.exit(0)

        txn = transaction.ApplicationUpdateTxn(
            index=self.coreid,
            sender=self.foundation.getAddress(),
            approval_program=b64decode(approval["result"]),
            clear_program=b64decode(clear["result"]),
            app_args=[ ],
            sp=self.client.suggested_params(),
        )
    
        signedTxn = txn.sign(self.foundation.getPrivateKey())
        print("sending transaction")
        self.client.send_transaction(signedTxn)
        resp = self.waitForTransaction(self.client, signedTxn.get_txid())
        pprint.pprint(resp.__dict__["logs"])
        print("complete")

    def updateToken(self) -> None:
        approval, clear = get_token_bridge(self.client, seed_amt=self.seed_amt, tmpl_sig=self.tsig)
        print("Updating the token contracts: " + str(len(b64decode(approval["result"]))))

        txn = transaction.ApplicationUpdateTxn(
            index=self.tokenid,
            sender=self.foundation.getAddress(),
            approval_program=b64decode(approval["result"]),
            clear_program=b64decode(clear["result"]),
            app_args=[ ],
            sp=self.client.suggested_params(),
        )
    
        signedTxn = txn.sign(self.foundation.getPrivateKey())
        print("sending transaction")
        self.client.send_transaction(signedTxn)
        resp = self.waitForTransaction(self.client, signedTxn.get_txid())
        for x in resp.__dict__["logs"]:
            print(x.hex())
        print("complete")

    def main(self) -> None:
        parser = argparse.ArgumentParser(description='algorand setup')
    
        parser.add_argument('--algod_address', type=str, help='algod address (default: http://localhost:4001)', 
                            default="http://localhost:4001")
        parser.add_argument('--algod_token', type=str, help='algod access token', 
                            default="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
        parser.add_argument('--kmd_address', type=str, help='kmd wallet address (default: http://localhost:4002)',
                            default="http://localhost:4002")
        parser.add_argument('--kmd_token', type=str, help='kmd wallet access token', 
                            default="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
        parser.add_argument('--kmd_name', type=str, help='kmd wallet name', 
                            default="unencrypted-default-wallet")
        parser.add_argument('--kmd_password', type=str, help='kmd wallet password', default="")

        parser.add_argument('--mnemonic', type=str, help='account mnemonic', default="")

        parser.add_argument('--coreid', type=int, help='core contract', default=4)
        parser.add_argument('--tokenid', type=int, help='token bridge contract', default=6)
        parser.add_argument('--devnet', action='store_true', help='setup devnet')
        parser.add_argument('--boot', action='store_true', help='bootstrap')
        parser.add_argument('--upgradePayload', action='store_true', help='gen the upgrade payload for the guardians to sign')
        parser.add_argument('--vaa', type=str, help='Submit the supplied VAA', default="")
        parser.add_argument('--submit', action='store_true', help='submit the synthetic vaas')
        parser.add_argument('--updateCore', action='store_true', help='update the Core contracts')
        parser.add_argument('--updateToken', action='store_true', help='update the Token contracts')
        parser.add_argument('--upgradeVAA', action='store_true', help='generate a upgrade vaa for devnet')
        parser.add_argument('--print', action='store_true', help='print')
    
        args = parser.parse_args()

        if args.devnet:
            self.init(args)

            if self.foundation == None:
                print("Generating the foundation account...")
                self.foundation = self.getTemporaryAccount(self.client)

            self.coreid = args.coreid
            self.tokenid = args.tokenid

            if args.upgradeVAA:
                ret = self.devnetUpgradeVAA()
                pprint.pprint(ret)
                if (args.submit) :
                    print("submitting vaa to upgrade core")
                    self.submitVAA(bytes.fromhex(ret[0]), self.client, self.foundation, self.coreid)
                    print("submitting vaa to upgrade token")
                    self.submitVAA(bytes.fromhex(ret[1]), self.client, self.foundation, self.tokenid)
    
            if args.upgradePayload:
                self.init(args)
                print(self.genUpgradePayload())
                sys.exit(0)
    
        if args.boot:
            self.dev_deploy()

        if args.updateCore:
            self.updateCore()

        if args.updateToken:
            self.updateToken()

core = PortalCore()
core.main()

# python3 admin.py --devnet --upgradeVAA --submit
