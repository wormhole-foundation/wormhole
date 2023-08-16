# python3 -m pip install pycryptodomex uvarint pyteal web3 coincurve

import os
from os.path import exists
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
from wormhole_core import getCoreContracts
from TmplSig import TmplSig
import argparse
from gentest import GenTest

from algosdk.v2client.algod import AlgodClient
from algosdk.kmd import KMDClient
from algosdk import account, mnemonic, abi
from algosdk.encoding import decode_address, encode_address
from algosdk.future import transaction
from pyteal import compileTeal, Mode, Expr
from pyteal import *
from algosdk.logic import get_application_address
from vaa_verify import get_vaa_verify

from Cryptodome.Hash import keccak

from algosdk.future.transaction import LogicSig

from token_bridge import get_token_bridge

from test_contract import get_test_app

from algosdk.v2client import indexer

import pprint

max_keys = 15
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
        print (privateKey)
        print ("    " + self.getMnemonic())
        print ("    " + self.addr)

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
        self.devnet = False

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

        self.myindexer = None

        self.seed_amt = int(1002000)  # The black magic in this number... 
        self.cache = {}
        self.asset_cache = {}

        self.kmdAccounts : Optional[List[Account]] = None

        self.accountList : List[Account] = []
        self.zeroPadBytes = "00"*32

        self.tsig = TmplSig("sig")


    def init(self, args) -> None:
        self.args = args
        self.ALGOD_ADDRESS = args.algod_address
        self.ALGOD_TOKEN = args.algod_token
        self.KMD_ADDRESS = args.kmd_address
        self.KMD_TOKEN = args.kmd_token
        self.KMD_WALLET_NAME = args.kmd_name
        self.KMD_WALLET_PASSWORD = args.kmd_password
        self.TARGET_ACCOUNT = args.mnemonic
        self.coreid = args.coreid
        self.tokenid = args.tokenid

        if exists(self.args.env):
            if self.gt == None:
                self.gt = GenTest(False)

            with open(self.args.env, encoding = 'utf-8') as f:
                for line in f:
                    e = line.rstrip('\n').split("=")
                    if "INIT_SIGNERS_CSV" in e[0]:
                        self.gt.guardianKeys = e[1].split(",")
                        print("guardianKeys=" + str(self.gt.guardianKeys))
                    if "INIT_SIGNERS_KEYS_CSV" in e[0]:
                        self.gt.guardianPrivKeys = e[1].split(",")
                        print("guardianPrivKeys=" + str(self.gt.guardianPrivKeys))

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


    def _fundFromGenesis(self, accountList, fundingAmt, client):        
            genesisAccounts = self.getGenesisAccounts()
            suggestedParams = client.suggested_params()
            txns: List[transaction.Transaction] = []
            for i, a in enumerate(accountList):
                fundingAccount = genesisAccounts[i % len(genesisAccounts)]
                txns.append(
                    transaction.PaymentTxn(
                        sender=fundingAccount.getAddress(),
                        receiver=a.getAddress(),
                        amt=fundingAmt,
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

    def getTemporaryAccount(self, client: AlgodClient) -> Account:
        if len(self.accountList) == 0:
            sks = [account.generate_account()[0] for i in range(3)]
            self.accountList = [Account(sk) for sk in sks]
            self._fundFromGenesis(self.accountList, self.FUNDING_AMOUNT, client)
    
        return self.accountList.pop()

    def fundDevAccounts(self, client: AlgodClient):
        devAcctsMnemonics = [
            "provide warfare better filter glory civil help jacket alpha penalty van fiber code upgrade web more curve sauce merit bike satoshi blame orphan absorb modify",
            "album neglect very nasty input trick annual arctic spray task candy unfold letter drill glove sword flock omit dial rather session mesh slow abandon slab",
            "blue spring teach silent cheap grace desk crack agree leave tray lady chair reopen midnight lottery glove congress lounge arrow fine junior mirror above purchase",
            "front rifle urge write push dynamic oil vital section blast protect suffer shoulder base address teach sight trap trial august mechanic border leaf absorb attract",
            "fat pet option agree father glue range ancient curtain pottery search raven club save crane sting gift seven butter decline image toward kidney above balance"
        ]

        accountList = []
        accountFunding = 400000000000000 # 400M algos

        for mnemo in devAcctsMnemonics:
            acc = Account.FromMnemonic(mnemo)
            print('Funding dev account {} with {} uALGOs'.format(acc.addr, accountFunding))
            accountList.append(acc)

        self._fundFromGenesis(accountList, accountFunding, client)

    
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
        results = self.client.application_info(app_id)
        return self.format_state(results['params']['global-state'])

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
        raise Exception("invalid type")

    def devnetUpgradeVAA(self):
        v = self.genUpgradePayload()
        print("core payload: " + str(v[0]))
        print("token payload: " + str(v[1]))

        if self.gt == None:
            self.gt = GenTest(False)

        emitter = bytes.fromhex(self.zeroPadBytes[0:(31*2)] + "04")

        guardianSet = self.getGovSet()

        print("guardianSet: " + str(guardianSet))

        nonce = int(random.random() * 20000)
        ret = [
            self.gt.createSignedVAA(guardianSet, self.gt.guardianPrivKeys, int(time.time()), nonce, 1, emitter, int(random.random() * 20000), 32, 8, v[0]),
            self.gt.createSignedVAA(guardianSet, self.gt.guardianPrivKeys, int(time.time()), nonce, 1, emitter, int(random.random() * 20000), 32, 8, v[1]),
        ]
        
#        pprint.pprint(self.parseVAA(bytes.fromhex(ret[0])))
#        pprint.pprint(self.parseVAA(bytes.fromhex(ret[1])))

        return ret

    def getMessageFee(self):
        s = self.client.application_info(self.coreid)["params"]["global-state"]
        k = base64.b64encode(b"MessageFee").decode('utf-8')
        for x in s:
            if x["key"] == k:
                return x["value"]["uint"]
        return -1

    def getGovSet(self):
        s = self.client.application_info(self.coreid)["params"]["global-state"]
        k = base64.b64encode(b"currentGuardianSetIndex").decode('utf-8')
        for x in s:
            if x["key"] == k:
                return x["value"]["uint"]
        return -1

    def genUpgradePayload(self):
        approval1, clear1 = getCoreContracts(False, self.args.core_approve, self.args.core_clear, self.client, seed_amt=self.seed_amt, tmpl_sig=self.tsig, devMode = self.devnet or self.args.testnet)

        approval2, clear2 = get_token_bridge(False, self.args.token_approve, self.args.token_clear, self.client, seed_amt=self.seed_amt, tmpl_sig=self.tsig, devMode = self.devnet or self.args.testnet)

        return self.genUpgradePayloadBody(approval1, approval2)

    def genUpgradePayloadBody(self, approval1, approval2):
        b  = self.zeroPadBytes[0:(28*2)]
        b += self.encoder("uint8", ord("C"))
        b += self.encoder("uint8", ord("o"))
        b += self.encoder("uint8", ord("r"))
        b += self.encoder("uint8", ord("e"))
        b += self.encoder("uint8", 1)
        b += self.encoder("uint16", 8)

        b += decode_address(approval1["hash"]).hex()
        print("core hash: " + decode_address(approval1["hash"]).hex())

        ret = [b]

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
        b += decode_address(approval2["hash"]).hex()
        print("token hash: " + decode_address(approval2["hash"]).hex())

        ret.append(b)
        return ret

    def createPortalCoreApp(
        self,
        client: AlgodClient,
        sender: Account,
    ) -> int:
        approval, clear = getCoreContracts(False, self.args.core_approve, self.args.core_clear, client, seed_amt=self.seed_amt, tmpl_sig=self.tsig, devMode = self.devnet or self.args.testnet)

        globalSchema = transaction.StateSchema(num_uints=8, num_byte_slices=40)
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
        approval, clear = get_token_bridge(False, self.args.token_approve, self.args.token_clear, client, seed_amt=self.seed_amt, tmpl_sig=self.tsig, devMode = self.devnet or self.args.testnet)

        if len(b64decode(approval["result"])) > 4060:
            print("token bridge contract is too large... This might prevent updates later")

        globalSchema = transaction.StateSchema(num_uints=4, num_byte_slices=30)
        localSchema = transaction.StateSchema(num_uints=0, num_byte_slices=16)
    
        app_args = [self.coreid, decode_address(get_application_address(self.coreid))]

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
        #pprint.pprint(response.__dict__)
        assert response.applicationIndex is not None and response.applicationIndex > 0

        # Lets give it a bit of money so that it is not a "ghost" account
        txn = transaction.PaymentTxn(sender = sender.getAddress(), sp = client.suggested_params(), receiver = get_application_address(response.applicationIndex), amt = 100000)
        signedTxn = txn.sign(sender.getPrivateKey())
        client.send_transaction(signedTxn)

        return response.applicationIndex

    def createTestApp(
        self,
        client: AlgodClient,
        sender: Account,
    ) -> int:
        approval, clear = get_test_app(client)

        globalSchema = transaction.StateSchema(num_uints=4, num_byte_slices=30)
        localSchema = transaction.StateSchema(num_uints=0, num_byte_slices=16)
    
        txn = transaction.ApplicationCreateTxn(
            sender=sender.getAddress(),
            on_complete=transaction.OnComplete.NoOpOC,
            approval_program=b64decode(approval["result"]),
            clear_program=b64decode(clear["result"]),
            global_schema=globalSchema,
            local_schema=localSchema,
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
                seed_txn.fee = seed_txn.fee * 2

                optin_txn = transaction.ApplicationOptInTxn(sig_addr, sp, app_id, rekey_to=get_application_address(app_id))
                optin_txn.fee = 0
    
                transaction.assign_group_id([seed_txn, optin_txn])
    
                signed_seed = seed_txn.sign(sender.getPrivateKey())
                signed_optin = transaction.LogicSigTransaction(optin_txn, lsa)
    
                client.send_transactions([signed_seed, signed_optin])
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

        sp = client.suggested_params()

        txns = [
            transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=coreid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"nop", b"0"],
                sp=sp
            ),

            transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=coreid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"nop", b"1"],
                sp=sp
            ),

            transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=coreid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"init", vaa, decode_address(self.vaa_verify["hash"])],
                accounts=[seq_addr, guardian_addr, newguardian_addr],
                sp=sp
            ),

            transaction.PaymentTxn(
                sender=sender.getAddress(),
                receiver=self.vaa_verify["hash"],
                amt=100000,
                sp=sp
            )
        ]

        return self.sendTxn(client, sender, txns, True)

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
                k = base64.b64decode(kv["key"])
                if k == "meta":
                    continue
                key = int.from_bytes(k, "big")
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
                amt = pmt
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

    def check_bits_set(self, client, app_id, addr, seq):
        bits_set = {}

        app_state = None
        ai = client.account_info(addr)
        for app in ai["apps-local-state"]:
            if app["id"] == app_id:
                app_state = app["key-value"]
        if app_state == None:
            return False

        start = int(seq / max_bits) * max_bits
        s = int((seq - start) / bits_per_key)
        b = int(((seq - start) - (s * bits_per_key)) / 8)

        k = base64.b64encode(s.to_bytes(1, "big")).decode('utf-8')
        for kv in app_state:
            if kv["key"] != k:
                continue
            v = base64.b64decode(kv["value"]["bytes"])
            bt = 1 << (seq%8)
            return ((v[b] & bt) != 0)

        return False

    def submitVAA(self, vaa, client, sender, appid):
        # A lot of our logic here depends on parseVAA and knowing what the payload is..
        p = self.parseVAA(vaa)

        #pprint.pprint(p)

        seq_addr = self.optin(client, sender, appid, int(p["sequence"] / max_bits), p["chainRaw"].hex() + p["emitter"].hex())

        # assert self.check_bits_set(client, appid, seq_addr, p["sequence"]) == False
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
        print("keys: " + keys.hex())

        sp = client.suggested_params()

        txns = []

        # How many signatures can we process in a single txn... we can do 9!
        bsize = (9*66)
        # audit: this was incorrectly adding an extra, empty block when the amount
        # of signatures was a multiple of 9. fixed.
        blocks = int(len(p["signatures"]) / bsize) + int(vaa[5] % 9 != 0)

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
            txns[-1].fee = 0

        txns.append(transaction.ApplicationCallTxn(
            sender=sender.getAddress(),
            index=self.coreid,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"verifyVAA", vaa],
            accounts=accts,
            sp=sp
        ))
        txns[-1].fee = txns[-1].fee * (1 + blocks)

        if p["Meta"] == "CoreGovernance":
            txns.append(transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=self.coreid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"governance", vaa],
                accounts=accts,
                sp=sp
            ))
            txns.append(transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=self.coreid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"nop", 5],
                sp=sp
            ))

        if p["Meta"] == "TokenBridge RegisterChain" or p["Meta"] == "TokenBridge UpgradeContract":
            txns.append(transaction.ApplicationCallTxn(
                sender=sender.getAddress(),
                index=self.tokenid,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"governance", vaa],
                accounts=accts,
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

            aid = 0

            if p["ToChain"] == 8 and p["Type"] == 3:
                aid = int.from_bytes(bytes.fromhex(p["ToAddress"]), "big")
                addr = get_application_address(aid)
            else:
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
                app_args=[b"completeTransfer", vaa],
                accounts=accts,
                foreign_assets = foreign_assets,
                sp=sp
            ))

            if aid != 0:
                txns[-1].foreign_apps = [aid]

            # We need to cover the inner transactions
            if p["Fee"] != self.zeroPadBytes:
                txns[-1].fee = txns[-1].fee * 3
            else:
                txns[-1].fee = txns[-1].fee * 2

            if p["Meta"] == "TokenBridge Transfer With Payload":
                m = abi.Method("portal_transfer", [abi.Argument("byte[]")], abi.Returns("byte[]"))
                txns.append(transaction.ApplicationCallTxn(
                    sender=sender.getAddress(),

                    index=int.from_bytes(bytes.fromhex(p["ToAddress"])[24:], "big"),
                    on_complete=transaction.OnComplete.NoOpOC,
                    app_args=[m.get_selector(), m.args[0].type.encode(vaa)],
                    foreign_assets = foreign_assets,
                    sp=sp
                ))

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

        # assert self.check_bits_set(client, appid, seq_addr, p["sequence"]) == True

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
        pprint.pprint((vaa[off:(off + 32)].hex(), "00000000000000000000000000000000000000000000000000000000436f7265"))
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

        if len(vaa[off:]) > 133 and int.from_bytes((vaa[off:off+1]), "big") == 3:
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
            ret["Fee"] = self.zeroPadBytes;
            ret["FromAddress"] = vaa[off:(off + 32)].hex()
            off += 32
            ret["Payload"] = vaa[off:].hex()
        
        return ret

    def boot(self):
        print("")

        print("Creating the PortalCore app")
        self.coreid = self.createPortalCoreApp(client=self.client, sender=self.foundation)
        pprint.pprint({"wormhole core": str(self.coreid), "address": get_application_address(self.coreid), "emitterAddress": decode_address(get_application_address(self.coreid)).hex()})

        print("Create the token bridge")
        self.tokenid = self.createTokenBridgeApp(self.client, self.foundation)
        pprint.pprint({"token bridge": str(self.tokenid), "address": get_application_address(self.tokenid), "emitterAddress": decode_address(get_application_address(self.tokenid)).hex()})

        if self.devnet or self.args.testnet:

            if self.devnet:
                print("Create test app")
                self.testid = self.createTestApp(self.client, self.foundation)
                pprint.pprint({"testapp": str(self.testid)})

                suggestedParams = self.client.suggested_params()
                fundingAccount = self.getGenesisAccounts()[0]

                txns: List[transaction.Transaction] = []
                wallet = "castle sing ice patrol mixture artist violin someone what access slow wrestle clap hero sausage oyster boost tone receive rapid bike announce pepper absent involve"
                a = Account.FromMnemonic(wallet)
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
                    txn.sign(fundingAccount.getPrivateKey()) for i, txn in enumerate(txns)
                ]
    
                self.client.send_transactions(signedTxns)
                print("Sent some ALGO to: " + wallet)

                print("Creating a Token...")
                txn = transaction.AssetConfigTxn(
                    sender=a.getAddress(),
                    sp=suggestedParams,
                    total=1000000,
                    default_frozen=False,
                    unit_name="NORIUM",
                    asset_name="ChuckNorium",
                    manager=a.getAddress(),
                    reserve=a.getAddress(),
                    freeze=a.getAddress(),
                    clawback=a.getAddress(),
                    strict_empty_address_check=False,
                    decimals=6)
                stxn = txn.sign(a.getPrivateKey())
                txid = self.client.send_transaction(stxn)
                print("NORIUM creation transaction ID: {}".format(txid))
                confirmed_txn = transaction.wait_for_confirmation(self.client, txid, 4)
                print("TXID: ", txid)
                print("Result confirmed in round: {}".format(confirmed_txn['confirmed-round']))

                print("Creating an NFT...")
                # JSON file
                dir_path = os.path.dirname(os.path.realpath(__file__))
                f = open (dir_path + 'cnNftMetadata.json', "r")
  
                # Reading from file
                metadataJSON = json.loads(f.read())
                metadataStr = json.dumps(metadataJSON)

                hash = hashlib.new("sha512_256")
                hash.update(b"arc0003/amj")
                hash.update(metadataStr.encode("utf-8"))
                json_metadata_hash = hash.digest()
                print("json_metadata_hash: ", hash.hexdigest())

				# Create transaction
                txn = transaction.AssetConfigTxn(
                    sender=a.getAddress(),
                    sp=suggestedParams,
                    total=1,
                    default_frozen=False,
                    unit_name="CNART",
                    asset_name="ChuckNoriumArtwork@arc3",
                    manager=a.getAddress(),
                    reserve=a.getAddress(),
                    freeze=a.getAddress(),
                    clawback=a.getAddress(),
                    strict_empty_address_check=False,
                    url="file://cnNftMetadata.json",
                    metadata_hash=json_metadata_hash,
                    decimals=0)
                stxn = txn.sign(a.getPrivateKey())
                txid = self.client.send_transaction(stxn)
                print("NORIUM NFT creation transaction ID: {}".format(txid))
                confirmed_txn = transaction.wait_for_confirmation(self.client, txid, 4)
                print("TXID: ", txid)
                print("Result confirmed in round: {}".format(confirmed_txn['confirmed-round']))

            if exists(self.args.env):
                if self.gt == None:
                    self.gt = GenTest(False)

                with open(self.args.env, encoding = 'utf-8') as f:
                    for line in f:
                        e = line.rstrip('\n').split("=")
                        print(e)
                        if "TOKEN_BRIDGE" in e[0]:
                            v = bytes.fromhex(e[1])
                            self.submitVAA(v, self.client, self.foundation, self.tokenid)
                        if "INIT_SIGNERS_CSV" in e[0]:
                            self.gt.guardianKeys = e[1].split(",")
                            print("guardianKeys: " + str(self.gt.guardianKeys))
                        if "INIT_SIGNERS_KEYS_CSV" in e[0]:
                            print("bootstrapping the guardian set...")
                            self.gt.guardianPrivKeys = e[1].split(",")
                            print("guardianPrivKeys: " + str(self.gt.guardianPrivKeys))

                            seq = int(random.random() * (2**31))
                            bootVAA = self.gt.genGuardianSetUpgrade(self.gt.guardianPrivKeys, self.args.guardianSet, self.args.guardianSet, seq, seq)
                            print("dev vaa: " + bootVAA)
                            self.bootGuardians(bytes.fromhex(bootVAA), self.client, self.foundation, self.coreid)
                seq = int(random.random() * (2**31))
                regChain = self.gt.genRegisterChain(self.gt.guardianPrivKeys, self.args.guardianSet, seq, seq, 8, decode_address(get_application_address(self.tokenid)).hex())
                print("ALGO_TOKEN_BRIDGE_VAA=" + regChain)
#                if self.args.env != ".env":
#                    v = bytes.fromhex(regChain)
#                    self.submitVAA(v, self.client, self.foundation, self.tokenid)
#                    print("We submitted it!")

    def updateCore(self) -> None:
        print("Updating the core contracts")
        if self.args.approve == "" and self.args.clear == "":
            approval, clear = getCoreContracts(False, self.args.core_approve, self.args.core_clear, self.client, seed_amt=self.seed_amt, tmpl_sig=self.tsig, devMode = self.devnet or self.args.testnet)
            print("core approval " + decode_address(approval["hash"]).hex())
            print("core clear " + decode_address(clear["hash"]).hex())
        else:
            pprint.pprint([self.args.approve, self.args.clear])
            with open(self.args.approve, encoding = 'utf-8') as f:
                approval = {"result": f.readlines()[0]}
                pprint.pprint(approval)
            with open(self.args.clear, encoding = 'utf-8') as f:
                clear = {"result": f.readlines()[0]}
                pprint.pprint(clear)

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
        pprint.pprint(resp)
        for x in resp.__dict__["logs"]:
            print(x.hex())
        print("complete")

    def updateToken(self) -> None:
        if self.args.approve == "" and self.args.clear == "":
            approval, clear = get_token_bridge(False, self.args.token_approve, self.args.token_clear, self.client, seed_amt=self.seed_amt, tmpl_sig=self.tsig, devMode = self.devnet or self.args.testnet)
        else:
            pprint.pprint([self.args.approve, self.args.clear])
            with open(self.args.approve, encoding = 'utf-8') as f:
                approval = {"result": f.readlines()[0]}
                pprint.pprint(approval)
            with open(self.args.clear, encoding = 'utf-8') as f:
                clear = {"result": f.readlines()[0]}
                pprint.pprint(clear)

#        print("token " + decode_address(approval["hash"]).hex())

        print("Updating the token contracts: " + str(len(b64decode(approval["result"]))))

        state = self.read_global_state(self.client, self.foundation.addr, self.tokenid)
        pprint.pprint( { 
            "validUpdateApproveHash": b64decode(state["validUpdateApproveHash"]).hex(),
            "validUpdateClearHash": b64decode(state["validUpdateClearHash"]).hex()
        })

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

    def genTeal(self) -> None:
        print((True, self.args.core_approve, self.args.core_clear, self.client, self.seed_amt, self.tsig, self.devnet or self.args.testnet))
        devmode = (self.devnet or self.args.testnet) and not self.args.prodTeal
        approval1, clear1 = getCoreContracts(True, self.args.core_approve, self.args.core_clear, self.client, seed_amt=self.seed_amt, tmpl_sig=self.tsig, devMode = devmode)
        print("Generating the teal for the core contracts")
        approval2, clear2 = get_token_bridge(True, self.args.token_approve, self.args.token_clear, self.client, seed_amt=self.seed_amt, tmpl_sig=self.tsig, devMode = devmode)
        print("Generating the teal for the token contracts: " + str(len(b64decode(approval2["result"]))))

        if self.devnet:
            v = self.genUpgradePayloadBody(approval1, approval2)
            if self.gt == None:
                self.gt = GenTest(False)
    
            emitter = bytes.fromhex(self.zeroPadBytes[0:(31*2)] + "04")
    
            guardianSet = 0
    
            nonce = int(random.random() * 20000)
            coreVAA = self.gt.createSignedVAA(guardianSet, self.gt.guardianPrivKeys, int(time.time()), nonce, 1, emitter, int(random.random() * 20000), 32, 8, v[0])
            tokenVAA = self.gt.createSignedVAA(guardianSet, self.gt.guardianPrivKeys, int(time.time()), nonce, 1, emitter, int(random.random() * 20000), 32, 8, v[1])
    
            with open("teal/core_devnet.vaa", "w") as fout:
                fout.write(coreVAA)
    
            with open("teal/token_devnet.vaa", "w") as fout:
                fout.write(tokenVAA)

    def testnet(self):
        self.ALGOD_ADDRESS = self.args.algod_address = "https://testnet-api.algonode.cloud"
        self.INDEXER_ADDRESS = "https://testnet-idx.algonode.cloud"
        self.coreid = self.args.coreid
        self.tokenid = self.args.tokenid

    def mainnet(self):
        self.ALGOD_ADDRESS = self.args.algod_address = "https://mainnet-api.algonode.cloud"
        self.INDEXER_ADDRESS = "https://mainnet-idx.algonode.cloud"
        self.coreid = 842125965
        self.tokenid = 842126029
        if self.args.coreid != 1004:
            self.coreid = self.args.coreid
        if self.args.tokenid != 1006:
            self.tokenid = self.args.tokenid

    def setup_args(self) -> None:
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
        
        parser.add_argument('--fundDevAccounts', action='store_true', help='Fund predetermined set of devnet accounts')

        parser.add_argument('--core_approve', type=str, help='core approve teal', default="teal/core_approve.teal")
        parser.add_argument('--core_clear', type=str, help='core clear teal', default="teal/core_clear.teal")

        parser.add_argument('--token_approve', type=str, help='token approve teal', default="teal/token_approve.teal")
        parser.add_argument('--token_clear', type=str, help='token clear teal', default="teal/token_clear.teal")

        parser.add_argument('--coreid', type=int, help='core contract', default=1004)
        parser.add_argument('--tokenid', type=int, help='token bridge contract', default=1006)
        parser.add_argument('--devnet', action='store_true', help='setup devnet')
        parser.add_argument('--boot', action='store_true', help='bootstrap')
        parser.add_argument('--upgradePayload', action='store_true', help='gen the upgrade payload for the guardians to sign')
        parser.add_argument('--vaa', type=str, help='Submit the supplied VAA', default="")
        parser.add_argument('--env', type=str, help='deploying using the supplied .env file', default=".env")
        parser.add_argument('--guardianSet', type=int, help='What guardianSet should I syntheticly create if needed', default=0)
        parser.add_argument('--appid', type=str, help='The appid that the vaa submit is applied to', default="")
        parser.add_argument('--submit', action='store_true', help='submit the synthetic vaas')
        parser.add_argument('--updateCore', action='store_true', help='update the Core contracts')
        parser.add_argument('--updateToken', action='store_true', help='update the Token contracts')
        parser.add_argument('--upgradeVAA', action='store_true', help='generate a upgrade vaa for devnet')
        parser.add_argument('--print', action='store_true', help='print')
        parser.add_argument('--genParts', action='store_true', help='Get tssig parts')
        parser.add_argument('--prodTeal', action='store_true', help='use Production Deal')
        parser.add_argument('--genTeal', action='store_true', help='Generate all the teal from the pyteal')
        parser.add_argument('--fund', action='store_true', help='Generate some accounts and fund them')
        parser.add_argument('--testnet', action='store_true', help='Connect to testnet')
        parser.add_argument('--mainnet', action='store_true', help='Connect to mainnet')
        parser.add_argument('--bootGuardian', type=str, help='Submit the supplied VAA', default="")
        parser.add_argument('--rpc', type=str, help='RPC address', default="")
        parser.add_argument('--guardianKeys', type=str, help='GuardianKeys', default="")
        parser.add_argument('--guardianPrivKeys', type=str, help='guardianPrivKeys', default="")
        parser.add_argument('--approve', type=str, help='compiled approve contract', default="")
        parser.add_argument('--clear', type=str, help='compiled clear contract', default="")

        parser.add_argument("--loops", type=int, help="testing: how many iterations should randomized tests run for. defaults to 1 for faster testing.", default="1")
        parser.add_argument("--bigset", action="store_true", help="testing: use the big set of validators", default="1")

        args = parser.parse_args()
        self.init(args)

        self.devnet = args.devnet

    def main(self) -> None:
        self.setup_args()

        args = self.args

        if args.testnet:
            self.testnet()

        if args.mainnet:
            self.mainnet()

        if args.rpc != "":
            self.ALGOD_ADDRESS = self.args.rpc
            
        self.client = self.getAlgodClient()

        if self.devnet or self.args.testnet:
            self.vaa_verify = self.client.compile(get_vaa_verify())
        else:
            c = AlgodClient("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "https://testnet-api.algonode.cloud")
            self.vaa_verify = c.compile(get_vaa_verify())

        self.vaa_verify["lsig"] = LogicSig(base64.b64decode(self.vaa_verify["result"]))

        if args.genTeal or args.boot:
            self.genTeal()
        
        # Generate the upgrade payload we need the guardians to sign
        if args.upgradePayload:
            print(self.genUpgradePayload())
            sys.exit(0)

        # This breaks the tsig up into the various parts so that we
        # can embed it into the Typescript code for reassembly  
        if args.genParts:
            print("this.ALGO_VERIFY_HASH = \"%s\""%self.vaa_verify["hash"]);
            print("this.ALGO_VERIFY = new Uint8Array([", end='')
            for x in b64decode(self.vaa_verify["result"]):
                print("%d, "%(x), end='')
            print("])")
    
            parts = [
                self.tsig.get_bytecode_raw(0).hex(),
                self.tsig.get_bytecode_raw(1).hex(),
                self.tsig.get_bytecode_raw(2).hex(),
                self.tsig.get_bytecode_raw(3).hex(),
                self.tsig.get_bytecode_raw(4).hex()
            ]

            pprint.pprint(parts)
            sys.exit(0)

        if args.mnemonic:
            self.foundation = Account.FromMnemonic(args.mnemonic)
        
        if args.devnet and self.foundation == None:
            print("Generating the foundation account...")
            self.foundation = self.getTemporaryAccount(self.client)
            print("Foundation account: " + self.foundation.getMnemonic())

        if self.args.fund:
            sys.exit(0)

        if self.foundation == None:
            print("We dont have a account?  Here is a random one I just made up...")
            pk = account.generate_account()[0]
            print(" pk:        " + pk)
            print(" address:   " + account.address_from_private_key(pk))
            print(" mnemonic:  " + mnemonic.from_private_key(pk))
            if args.testnet:
                print("go to https://bank.testnet.algorand.network/ to fill it up  (You will probably want to send at least 2 loads to the wallet)")
            sys.exit(0)

        bal = self.getBalances(self.client, self.foundation.addr)
        print("foundation address " + self.foundation.addr + "  (" + str(float(bal[0]) / 1000000.0) + " ALGO)")
        if bal[0] < 10000000:
            print("you need at least 10 ALGO to do darn near anything...")
            sys.exit(0)

        if args.guardianKeys != "":
            self.gt.guardianKeys = eval(args.guardianKeys)

        if args.guardianPrivKeys != "":
            self.gt.guardianPrivKeyss = eval(args.guardianPrivKeys)

        if args.upgradeVAA:
            ret = self.devnetUpgradeVAA()
            pprint.pprint(ret)
            if (args.submit) :
                print("submitting vaa to upgrade core: " + str(self.coreid))
                state = self.read_global_state(self.client, self.foundation.addr, self.coreid)
                pprint.pprint( { 
                    "validUpdateApproveHash": b64decode(state["validUpdateApproveHash"]).hex(),
                    "validUpdateClearHash": b64decode(state["validUpdateClearHash"]).hex()
                })
                self.submitVAA(bytes.fromhex(ret[0]), self.client, self.foundation, self.coreid)
                state = self.read_global_state(self.client, self.foundation.addr, self.coreid)
                pprint.pprint( { 
                    "validUpdateApproveHash": b64decode(state["validUpdateApproveHash"]).hex(),
                    "validUpdateClearHash": b64decode(state["validUpdateClearHash"]).hex()
                })

                print("submitting vaa to upgrade token: " + str(self.tokenid))
                state = self.read_global_state(self.client, self.foundation.addr, self.tokenid)
                pprint.pprint( { 
                    "validUpdateApproveHash": b64decode(state["validUpdateApproveHash"]).hex(),
                    "validUpdateClearHash": b64decode(state["validUpdateClearHash"]).hex()
                })
                self.submitVAA(bytes.fromhex(ret[1]), self.client, self.foundation, self.tokenid)
                state = self.read_global_state(self.client, self.foundation.addr, self.tokenid)
                pprint.pprint( { 
                    "validUpdateApproveHash": b64decode(state["validUpdateApproveHash"]).hex(),
                    "validUpdateClearHash": b64decode(state["validUpdateClearHash"]).hex()
                })

        if args.boot:
            self.boot()

        if args.updateCore:
            self.updateCore()

        if args.updateToken:
            self.updateToken()

        if args.vaa:
            if self.args.appid == "":
                raise Exception("You need to specifiy the appid when you are submitting vaas")
            vaa = bytes.fromhex(args.vaa)
            pprint.pprint(self.parseVAA(vaa))
            self.submitVAA(vaa, self.client, self.foundation, int(self.args.appid))

        if args.bootGuardian != "":
            vaa = bytes.fromhex(args.bootGuardian)
            pprint.pprint(self.parseVAA(vaa))
            response = self.bootGuardians(vaa, self.client, self.foundation, self.coreid)
            pprint.pprint(response.__dict__)

        if args.fundDevAccounts:
            if not args.devnet:
                print("Missing required parameter: --devnet")
                sys.exit(0)
            
            self.fundDevAccounts(self.client)
            
if __name__ == "__main__":
    core = PortalCore()
    core.main()
