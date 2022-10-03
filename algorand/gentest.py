from eth_abi import encode_single, encode_abi
import sys
import string
import pprint
import time
from Cryptodome.Hash import keccak
import coincurve
import base64
import random
from algosdk.encoding import decode_address

class GenTest:
    def __init__(self, bigSet) -> None:
        if bigSet:
            self.guardianKeys = [
                "52A26Ce40F8CAa8D36155d37ef0D5D783fc614d2",
                "389A74E8FFa224aeAD0778c786163a7A2150768C",
                "B4459EA6482D4aE574305B239B4f2264239e7599",
                "072491bd66F63356090C11Aae8114F5372aBf12B",
                "51280eA1fd2B0A1c76Ae29a7d54dda68860A2bfF",
                "fa9Aa60CfF05e20E2CcAA784eE89A0A16C2057CB",
                "e42d59F8FCd86a1c5c4bA351bD251A5c5B05DF6A",
                "4B07fF9D5cE1A6ed58b6e9e7d6974d1baBEc087e",
                "c8306B84235D7b0478c61783C50F990bfC44cFc0",
                "C8C1035110a13fe788259A4148F871b52bAbcb1B",
                "58A2508A20A7198E131503ce26bBE119aA8c62b2",
                "8390820f04ddA22AFe03be1c3bb10f4ba6CF94A0",
                "1FD6e97387C34a1F36DE0f8341E9D409E06ec45b",
                "255a41fC2792209CB998A8287204D40996df9E54",
                "bA663B12DD23fbF4FbAC618Be140727986B3BBd0",
                "79040E577aC50486d0F6930e160A5C75FD1203C6",
                "3580D2F00309A9A85efFAf02564Fc183C0183A96",
                "3869795913D3B6dBF3B24a1C7654672c69A23c35",
                "1c0Cc52D7673c52DE99785741344662F5b2308a0",
            ]

            self.guardianPrivKeys = [
                "563d8d2fd4e701901d3846dee7ae7a92c18f1975195264d676f8407ac5976757",
                "8d97f25916a755df1d9ef74eb4dbebc5f868cb07830527731e94478cdc2b9d5f",
                "9bd728ad7617c05c31382053b57658d4a8125684c0098f740a054d87ddc0e93b",
                "5a02c4cd110d20a83a7ce8d1a2b2ae5df252b4e5f6781c7855db5cc28ed2d1b4",
                "93d4e3b443bf11f99a00901222c032bd5f63cf73fc1bcfa40829824d121be9b2",
                "ea40e40c63c6ff155230da64a2c44fcd1f1c9e50cacb752c230f77771ce1d856",
                "87eaabe9c27a82198e618bca20f48f9679c0f239948dbd094005e262da33fe6a",
                "61ffed2bff38648a6d36d6ed560b741b1ca53d45391441124f27e1e48ca04770",
                "bd12a242c6da318fef8f98002efb98efbf434218a78730a197d981bebaee826e",
                "20d3597bb16525b6d09e5fb56feb91b053d961ab156f4807e37d980f50e71aff",
                "344b313ffbc0199ff6ca08cacdaf5dc1d85221e2f2dc156a84245bd49b981673",
                "848b93264edd3f1a521274ca4da4632989eb5303fd15b14e5ec6bcaa91172b05",
                "c6f2046c1e6c172497fc23bd362104e2f4460d0f61984938fa16ef43f27d93f6",
                "693b256b1ee6b6fb353ba23274280e7166ab3be8c23c203cc76d716ba4bc32bf",
                "13c41508c0da03018d61427910b9922345ced25e2bbce50652e939ee6e5ea56d",
                "460ee0ee403be7a4f1eb1c63dd1edaa815fbaa6cf0cf2344dcba4a8acf9aca74",
                "b25148579b99b18c8994b0b86e4dd586975a78fa6e7ad6ec89478d7fbafd2683",
                "90d7ac6a82166c908b8cf1b352f3c9340a8d1f2907d7146fb7cd6354a5436cca",
                "b71d23908e4cf5d6cd973394f3a4b6b164eb1065785feee612efdfd8d30005ed",
            ]

        else:
            self.guardianKeys = [
                "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
            ]
    
            self.guardianPrivKeys = [
                "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0"
            ]

        self.zeroPadBytes = "00"*64

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

    def createTrashVAA(self, guardianSetIndex, ts, nonce, emitterChainId, emitterAddress, sequence, consistencyLevel, target, payload, version=1):
        return self.createSignedVAA(
            guardianSetIndex,
            # set the minimum amount of trash as signature for this to pass validations
            [random.randbytes(32).hex() for _ in range(int(len(self.guardianKeys)*2/3)+1)],
            ts,
            nonce,
            emitterChainId,
            emitterAddress,
            sequence,
            consistencyLevel,
            target,
            payload,
            version
        )


    def createSignedVAA(self, guardianSetIndex, signers, ts, nonce, emitterChainId, emitterAddress, sequence, consistencyLevel, target, payload, version=1):
        b = ""

        b += self.encoder("uint32", ts)
        b += self.encoder("uint32", nonce)
        b += self.encoder("uint16", emitterChainId)
        b += self.encoder("bytes32", emitterAddress)
        b += self.encoder("uint64", sequence)
        b += self.encoder("uint8", consistencyLevel)
        b += payload

        hash = keccak.new(digest_bits=256).update(keccak.new(digest_bits=256).update(bytes.fromhex(b)).digest()).digest()

        signatures = ""

        for  i in range(len(signers)):
            signatures += self.encoder("uint8", i)

            key = coincurve.PrivateKey(bytes.fromhex(signers[i]))
            signature = key.sign_recoverable(hash, hasher=None)
            signatures += signature.hex()

        ret  = self.encoder("uint8", version)
        ret += self.encoder("uint32", guardianSetIndex)
        ret += self.encoder("uint8", len(signers))
        ret += signatures
        ret += b

        print(ret)
        return ret

    def createValidRandomSignedVAA(self, guardianSetIndex, signers, sequence):
        ts = random.randint(0, 2**32-1)
        nonce = random.randint(0, 2**32-1)
        emitterChainId = random.randint(0, 2**16-1)
        emitterAddress = random.randbytes(32)
        consitencyLevel = random.randint(0, 2**8-1)
        payload = self.createRandomValidPayload().hex()

        return self.createSignedVAA(
            guardianSetIndex, # guardian set index needs to be fixed so contract knows where to look into
            signers,
            ts,
            nonce,
            emitterChainId,
            emitterAddress,
            sequence,
            consitencyLevel,
            0, #target = not used?
            payload,
            1, # only version 1 VAA
        )

    def createRandomValidPayload(self):
        action = (0x03).to_bytes(1, byteorder="big")
        # action = random.choice([0x01, 0x03]).to_bytes(1, byteorder="big")
        amount = random.randint(0, 2**128-1).to_bytes(32, byteorder="big")

        # TODO: we should support more addresses than this one, but this
        # is hardcoded in the tests and probably used in the deploy, so we
        # will make do. same goes for the token_address
        some_token_address = b"4523c3F29447d1f32AEa95BEBD00383c4640F1b4"
        tokenAddress = some_token_address
        # TODO: same goes for the token chain, just use what's available for now
        try:
            tokenChain = bytes.fromhex(self.getEmitter(1))
        except:
            raise
        to = random.randbytes(32)
        toChain = random.randint(0, 2**16-1).to_bytes(2, byteorder="big")

        payload = action + amount + tokenAddress + tokenChain + to + toChain

        if action == 0x01:
            fee = random.randint(0, 2**256-1).to_bytes(32, byteorder="big")
            payload += fee

        if action == 0x03:
            fromAddress = random.randbytes(2)
            arbitraryPayload = random.randbytes(random.randint(0,4))
            payload += fromAddress + arbitraryPayload

        return payload



    def createRandomSignedVAA(self, guardianSetIndex, signers):
        ts = random.randint(0, 2**32-1)
        nonce = random.randint(0, 2**32-1)
        emitterChainId = random.randint(0, 2**16-1)
        emitterAddress = random.randbytes(32)
        sequence = random.randint(0, 2**64-1)
        consitencyLevel = random.randint(0, 2**8-1)
        # payload = ''.join(random.choices(string.ascii_uppercase + string.digits, k=random.randint(0,500)))
        payload = random.randbytes(random.randint(0,496)).hex()

        version = random.randint(0,10)

        return self.createSignedVAA(
            guardianSetIndex, # guardian set index needs to be fixed so contract knows where to look into
            signers,
            ts,
            nonce,
            emitterChainId,
            emitterAddress,
            sequence,
            consitencyLevel,
            0, #target = not used?
            payload,
            version,
        )

    def genGuardianSetUpgrade(self, signers, guardianSet, targetSet, nonce, seq):
        b  = self.zeroPadBytes[0:(28*2)]
        b += self.encoder("uint8", ord("C"))
        b += self.encoder("uint8", ord("o"))
        b += self.encoder("uint8", ord("r"))
        b += self.encoder("uint8", ord("e"))
        b += self.encoder("uint8", 2)
        b += self.encoder("uint16", 0)
        b += self.encoder("uint32", targetSet)
        b += self.encoder("uint8", len(self.guardianKeys))

        for i in self.guardianKeys:
            b += i

        emitter = bytes.fromhex(self.zeroPadBytes[0:(31*2)] + "04")
        return self.createSignedVAA(guardianSet, signers, int(time.time()), nonce, 1, emitter, seq, 32, 0, b)

    def genGSetFee(self, signers, guardianSet, nonce, seq, amt):
        b  = self.zeroPadBytes[0:(28*2)]
        b += self.encoder("uint8", ord("C"))
        b += self.encoder("uint8", ord("o"))
        b += self.encoder("uint8", ord("r"))
        b += self.encoder("uint8", ord("e"))
        b += self.encoder("uint8", 3)
        b += self.encoder("uint16", 8)
        b += self.encoder("uint256", int(amt))  # a whole algo!

        emitter = bytes.fromhex(self.zeroPadBytes[0:(31*2)] + "04")
        return self.createSignedVAA(guardianSet, signers, int(time.time()), nonce, 1, emitter, seq, 32, 0, b)

    def genGFeePayout(self, signers, guardianSet, targetSet, nonce, seq, amt, dest):
        b  = self.zeroPadBytes[0:(28*2)]
        b += self.encoder("uint8", ord("C"))
        b += self.encoder("uint8", ord("o"))
        b += self.encoder("uint8", ord("r"))
        b += self.encoder("uint8", ord("e"))
        b += self.encoder("uint8", 4)
        b += self.encoder("uint16", 8)
        b += self.encoder("uint256", int(amt * 1000000))
        b += decode_address(dest).hex()

        emitter = bytes.fromhex(self.zeroPadBytes[0:(31*2)] + "04")
        return self.createSignedVAA(guardianSet, signers, int(time.time()), nonce, 1, emitter, seq, 32, 0, b)

    def getEmitter(self, chain):
        if chain == 1:
            return "ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5"
        if chain == 2:
            return "0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"
        if chain == 3:
            return "0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2"
        if chain == 4:
            return "000000000000000000000000b6f6d86a8f9879a9c87f643768d9efc38c1da6e7"
        if chain == 5:
            return "0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde"
        raise Exception("invalid chain")
        
    def genRegisterChain(self, signers, guardianSet, nonce, seq, chain, addr = None):
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

        b += self.encoder("uint8", 1)  # action
        b += self.encoder("uint16", 0) # target chain
        b += self.encoder("uint16", chain)
        if addr == None:
            b += self.getEmitter(chain)
        else:
            b += addr
        emitter = bytes.fromhex(self.zeroPadBytes[0:(31*2)] + "04")
        return self.createSignedVAA(guardianSet, signers, int(time.time()), nonce, 1, emitter, seq, 32, 0, b)

    def genAssetMeta(self, signers, guardianSet, nonce, seq, tokenAddress, chain, decimals, symbol, name):
        b  = self.encoder("uint8", 2)
        b += self.zeroPadBytes[0:((32-len(tokenAddress))*2)]
        b += tokenAddress.hex()
        b += self.encoder("uint16", chain)
        b += self.encoder("uint8", decimals)
        b += symbol.hex()
        b += self.zeroPadBytes[0:((32-len(symbol))*2)]
        b += name.hex()
        b += self.zeroPadBytes[0:((32-len(name))*2)]
        emitter = bytes.fromhex(self.getEmitter(chain))
        return self.createSignedVAA(guardianSet, signers, int(time.time()), nonce, 1, emitter, seq, 32, 0, b)

    def genRandomValidTransfer(self,
                               signers,
                               guardianSet,
                               seq,
                               tokenAddress,
                               toAddress,
                               amount_max):
        amount = random.randint(0, int(amount_max / 100000000))
        fee = random.randint(0, amount) # fee must be lower than amount for VAA to be valid
        return self.genTransfer(
            signers=signers,
            guardianSet=guardianSet,
            nonce=random.randint(0, 2**32-1),
            seq=seq,
            # amount gets encoded as an uint256, but it's actually clearly
            # to only eight bytes. all other bytes _must_ be zero.
            amount=amount,
            # token address must be registed on the bridge
            tokenAddress=tokenAddress,
            # tokenAddress=random.randbytes(32),
            tokenChain=1,
            toAddress=toAddress,
            # must be directed at algorand chain
            toChain=8,
            # fee is in the same situation as amount
            fee=fee,
        )


    def genTransfer(self, signers, guardianSet, nonce, seq, amount, tokenAddress, tokenChain, toAddress, toChain, fee):
        b  = self.encoder("uint8", 1)
        b += self.encoder("uint256", int(amount * 100000000))

        b += self.zeroPadBytes[0:((32-len(tokenAddress))*2)]
        b += tokenAddress.hex()

        b += self.encoder("uint16", tokenChain)

        b += self.zeroPadBytes[0:((32-len(toAddress))*2)]
        b += toAddress.hex()

        b += self.encoder("uint16", toChain)

        b += self.encoder("uint256", int(fee * 100000000))

        emitter = bytes.fromhex(self.getEmitter(tokenChain))
        return self.createSignedVAA(guardianSet, signers, int(time.time()), nonce, 1, emitter, seq, 32, 0, b)

    def genVaa(self, emitter, seq, payload):
        nonce = int(random.random() * 4000000.0)
        return self.createSignedVAA(1, self.guardianPrivKeys, int(time.time()), nonce, 8, emitter, seq, 32, 0, payload.hex())

    def test(self):
        print(self.genTransfer(self.guardianPrivKeys, 1, 1, 1, 1, bytes.fromhex("4523c3F29447d1f32AEa95BEBD00383c4640F1b4"), 1, decode_address("ROOKEPZMHHBAEH75Y44OCNXQAGTXZWG3PY7IYQQCMXO7IG7DJMVHU32YVI"), 8, 0))
        
if __name__ == '__main__':    
    core = GenTest(True)
    core.test()
