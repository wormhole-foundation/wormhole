import sys

from terra_sdk.client.lcd import AsyncLCDClient
from terra_sdk.client.localterra import AsyncLocalTerra
from terra_sdk.core.auth import StdFee
import asyncio
from terra_sdk.core.wasm import (
    MsgStoreCode,
    MsgInstantiateContract,
    MsgExecuteContract,
    MsgMigrateContract,
)
from terra_sdk.key.mnemonic import MnemonicKey
from terra_sdk.util.contract import get_code_id, get_contract_address, read_file_as_b64
import os
import base64
import pprint

if len(sys.argv) != 8:
    print(
        "Usage: deploy.py <lcd_url> <chain_id> <mnemonic> <gov_chain> <gov_address> <initial_guardian> <expiration_time>")
    exit(1)

gas_prices = {
    "uluna": "0.15",
    "usdr": "0.1018",
    "uusd": "0.15",
    "ukrw": "178.05",
    "umnt": "431.6259",
    "ueur": "0.125",
    "ucny": "0.97",
    "ujpy": "16",
    "ugbp": "0.11",
    "uinr": "11",
    "ucad": "0.19",
    "uchf": "0.13",
    "uaud": "0.19",
    "usgd": "0.2",
}

lt = AsyncLocalTerra(gas_prices={"uusd": "0.15"}, url="http://terra-lcd:1317")
terra = AsyncLCDClient(
    sys.argv[1], sys.argv[2], gas_prices=gas_prices
)
deployer = terra.wallet(MnemonicKey(
    mnemonic=sys.argv[3]))

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
    GOV_CHAIN = int(sys.argv[4])
    GOV_ADDRESS = bytes.fromhex(sys.argv[5])

    wormhole = await Contract.create(
        code_id=code_ids["wormhole"],
        gov_chain=GOV_CHAIN,
        gov_address=base64.b64encode(GOV_ADDRESS).decode("utf-8"),
        guardian_set_expirity=int(sys.argv[7]),
        initial_guardian_set={
            "addresses": [{"bytes": base64.b64encode(
                bytearray.fromhex(sys.argv[6])).decode("utf-8")}],
            "expiration_time": 0},
        migratable=True,
    )
    print("Wormhole contract: {}".format(wormhole.address))

    token_bridge = await Contract.create(
        code_id=code_ids["token_bridge"],
        owner=deployer.key.acc_address,
        gov_chain=GOV_CHAIN,
        gov_address=base64.b64encode(GOV_ADDRESS).decode("utf-8"),
        wormhole_contract=wormhole,
        wrapped_asset_code_id=int(code_ids["cw20_wrapped"]),
    )
    print("Token Bridge contract: {}".format(token_bridge.address))

    mock_token = await Contract.create(
        code_id=code_ids["cw20_base"],
        name="MOCK",
        symbol="MCK",
        decimals=6,
        initial_balances=[{"address": deployer.key.acc_address, "amount": "100000000"}],
        mint=None,
    )
    print("Example Token contract: {}".format(mock_token.address))


if __name__ == "__main__":
    asyncio.get_event_loop().run_until_complete(main())
