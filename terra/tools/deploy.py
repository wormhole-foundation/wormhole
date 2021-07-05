from terra_sdk.client.localterra import AsyncLocalTerra
from terra_sdk.core.auth import StdFee
import asyncio
from terra_sdk.core.wasm import (
    MsgStoreCode,
    MsgInstantiateContract,
    MsgExecuteContract,
    MsgMigrateContract,
)
from terra_sdk.util.contract import get_code_id, get_contract_address, read_file_as_b64
import os
import base64
import pprint

lt = AsyncLocalTerra(gas_prices={"uusd": "0.15"}, url="http://terra-lcd:1317")
terra = lt
deployer = lt.wallets["test1"]

sequence = asyncio.get_event_loop().run_until_complete(deployer.sequence())


async def sign_and_broadcast(*msgs):
    global sequence
    try:
        tx = await deployer.create_and_sign_tx(
            msgs=msgs, fee=StdFee(30000000, "20000000uusd"), sequence=sequence
        )
        result = await terra.tx.broadcast(tx)
        sequence += 1
        if result.is_tx_error():
            raise Exception(result.raw_log)
        return result
    except:
        sequence = await deployer.sequence()
        raise


async def store_contract(contract_name):
    parent_dir = os.path.dirname(__file__)
    contract_bytes = read_file_as_b64(f"{parent_dir}/../artifacts/{contract_name}.wasm")
    store_code = MsgStoreCode(deployer.key.acc_address, contract_bytes)

    result = await sign_and_broadcast(store_code)
    code_id = get_code_id(result)
    print(f"Code id for {contract_name} is {code_id}")
    return code_id


async def store_contracts():
    parent_dir = os.path.dirname(__file__)
    contract_names = [
        i[:-5] for i in os.listdir(f"{parent_dir}/../artifacts") if i.endswith(".wasm")
    ]

    return {
        contract_name: await store_contract(contract_name)
        for contract_name in contract_names
    }


class ContractQuerier:
    def __init__(self, address):
        self.address = address

    def __getattr__(self, item):
        async def result_fxn(**kwargs):
            kwargs = convert_contracts_to_addr(kwargs)
            return await terra.wasm.contract_query(self.address, {item: kwargs})

        return result_fxn


class Contract:
    @staticmethod
    async def create(code_id, migratable=False, **kwargs):
        kwargs = convert_contracts_to_addr(kwargs)
        instantiate = MsgInstantiateContract(
            deployer.key.acc_address, code_id, kwargs, migratable=migratable
        )
        result = await sign_and_broadcast(instantiate)
        return Contract(get_contract_address(result))

    def __init__(self, address):
        self.address = address

    def __getattr__(self, item):
        async def result_fxn(coins=None, **kwargs):
            kwargs = convert_contracts_to_addr(kwargs)
            execute = MsgExecuteContract(
                deployer.key.acc_address, self.address, {item: kwargs}, coins=coins
            )
            return await sign_and_broadcast(execute)

        return result_fxn

    @property
    def query(self):
        return ContractQuerier(self.address)

    async def migrate(self, new_code_id):
        migrate = MsgMigrateContract(
            contract=self.address,
            migrate_msg={},
            new_code_id=new_code_id,
            owner=deployer.key.acc_address,
        )
        return await sign_and_broadcast(migrate)


def convert_contracts_to_addr(obj):
    if type(obj) == dict:
        return {k: convert_contracts_to_addr(v) for k, v in obj.items()}
    if type(obj) in {list, tuple}:
        return [convert_contracts_to_addr(i) for i in obj]
    if type(obj) == Contract:
        return obj.address
    return obj


def to_bytes(n, length, byteorder="big"):
    return int(n).to_bytes(length, byteorder=byteorder)


def assemble_vaa(emitter_chain, emitter_address, payload):
    import time

    # version, guardian set index, len signatures
    header = to_bytes(1, 1) + to_bytes(0, 4) + to_bytes(0, 1)
    # timestamp, nonce, emitter_chain
    body = to_bytes(time.time(), 8) + to_bytes(1, 4) + to_bytes(emitter_chain, 2)
    # emitter_address, vaa payload
    body += emitter_address + payload

    return header + body


async def main():
    code_ids = await store_contracts()
    print(code_ids)

    # fake governance contract on solana
    GOV_CHAIN = 1
    GOV_ADDRESS = b"0" * 32

    wormhole = await Contract.create(
        code_id=code_ids["wormhole"],
        gov_chain=GOV_CHAIN,
        gov_address=base64.b64encode(GOV_ADDRESS).decode("utf-8"),
        guardian_set_expirity=10 ** 15,
        initial_guardian_set={
            "addresses": [{"bytes": base64.b64encode(
                bytearray.fromhex("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe")).decode("utf-8")}],
            "expiration_time": 0},
        migratable=True,
    )
    print("Wormhole contract: {}".format(wormhole.address))


if __name__ == "__main__":
    asyncio.get_event_loop().run_until_complete(main())
