from terra_sdk.client.localterra import AsyncLocalTerra
from terra_sdk.core.auth import StdFee
import asyncio
from terra_sdk.core.wasm import (
    MsgStoreCode,
    MsgInstantiateContract,
    MsgExecuteContract,
)
from terra_sdk.util.contract import get_code_id, get_contract_address, read_file_as_b64
import os
import base64
import pprint


lt = AsyncLocalTerra(gas_prices={"uusd": "0.15"})
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
    contract_bytes = read_file_as_b64(f"{parent_dir}/artifacts/{contract_name}.wasm")
    store_code = MsgStoreCode(deployer.key.acc_address, contract_bytes)

    result = await sign_and_broadcast(store_code)
    code_id = get_code_id(result)
    print(f"Code id for {contract_name} is {code_id}")
    return code_id


async def store_contracts():

    parent_dir = os.path.dirname(__file__)
    contract_names = [
        i[:-5] for i in os.listdir(f"{parent_dir}/artifacts") if i.endswith(".wasm")
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
    async def create(code_id, **kwargs):
        kwargs = convert_contracts_to_addr(kwargs)
        instantiate = MsgInstantiateContract(deployer.key.acc_address, code_id, kwargs)
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
    wormhole = await Contract.create(
        code_id=code_ids["wormhole"],
        guardian_set_expirity=10 ** 15,
        initial_guardian_set={"addresses": [], "expiration_time": 10 ** 15},
    )

    token_bridge = await Contract.create(
        code_id=code_ids["token_bridge"],
        owner=deployer.key.acc_address,
        wormhole_contract=wormhole,
        wrapped_asset_code_id=int(code_ids["cw20_wrapped"]),
    )

    mock_token = await Contract.create(
        code_id=code_ids["cw20_base"],
        name="MOCK",
        symbol="MCK",
        decimals=6,
        initial_balances=[{"address": deployer.key.acc_address, "amount": "100000000"}],
        mint=None,
    )

    raw_addr = deployer.key.raw_address
    recipient = b"\0" * 12 + raw_addr
    recipient = base64.b64encode(recipient)

    print(
        "Balance before initiate transfer",
        await mock_token.query.balance(address=deployer.key.acc_address),
    )

    await mock_token.increase_allowance(spender=token_bridge, amount="1000")
    bridge_canonical = bytes.fromhex(
        (await wormhole.query.query_address_hex(address=token_bridge))["hex"]
    )
    await token_bridge.register_chain(
        chain_id=3, chain_address=base64.b64encode(bridge_canonical).decode("utf-8")
    )

    resp = await token_bridge.initiate_transfer(
        asset=mock_token,
        amount="1000",
        recipient_chain=3,
        recipient=recipient.decode("utf-8"),
        nonce=0,
        coins={"uluna": "10000"},
    )

    print(
        "Balance after initiate transfer",
        await mock_token.query.balance(address=deployer.key.acc_address),
    )

    logs = resp.logs[0].events_by_type
    transfer_data = {
        k: v[0] for k, v in logs["from_contract"].items() if k.startswith("message")
    }
    vaa = assemble_vaa(
        transfer_data["message.chain_id"],
        bytes.fromhex(transfer_data["message.sender"]),
        bytes.fromhex(transfer_data["message.message"]),
    )

    await token_bridge.submit_vaa(data=base64.b64encode(vaa).decode("utf-8"))

    print(
        "Balance after complete transfer",
        await mock_token.query.balance(address=deployer.key.acc_address),
    )

    # pretend there exists another bridge contract with the same address but on solana
    await token_bridge.register_chain(
        chain_id=1, chain_address=base64.b64encode(bridge_canonical).decode("utf-8")
    )

    resp = await token_bridge.create_asset_meta(
        asset_address=mock_token,
        nonce=1,
        coins={"uluna": "10000"},
    )

    logs = resp.logs[0].events_by_type
    create_meta_data = {
        k: v[0] for k, v in logs["from_contract"].items() if k.startswith("message")
    }
    message_bytes = bytes.fromhex(create_meta_data["message.message"])

    # switch the chain of the asset meta to say its from solana
    message_bytes = message_bytes[:1] + to_bytes(1, 2) + message_bytes[3:]
    vaa = assemble_vaa(
        1,  # totally came from solana
        bytes.fromhex(create_meta_data["message.sender"]),
        message_bytes,
    )

    # attest this metadata and make a wrapped asset from solana
    resp = await token_bridge.submit_vaa(data=base64.b64encode(vaa).decode("utf-8"))

    wrapped_token = Contract(get_contract_address(resp))

    # now send from solana...
    message_bytes = bytes.fromhex(transfer_data["message.message"])
    message_bytes = message_bytes[:1] + to_bytes(1, 2) + message_bytes[3:]

    vaa = assemble_vaa(
        1,  # totally came from solana
        bytes.fromhex(transfer_data["message.sender"]),
        message_bytes,
    )
    print(
        "Balance before completing transfer from solana",
        await wrapped_token.query.balance(address=deployer.key.acc_address),
    )

    await token_bridge.submit_vaa(data=base64.b64encode(vaa).decode("utf-8"))

    print(
        "Balance after completing transfer from solana",
        await wrapped_token.query.balance(address=deployer.key.acc_address),
    )

    await wrapped_token.increase_allowance(spender=token_bridge, amount="1000")
    resp = await token_bridge.initiate_transfer(
        asset=wrapped_token,
        amount="1000",
        recipient_chain=1,
        recipient=recipient.decode("utf-8"),
        nonce=0,
        coins={"uluna": "10000"},
    )

    print(
        "Balance after completing transfer to solana",
        await wrapped_token.query.balance(address=deployer.key.acc_address),
    )


if __name__ == "__main__":
    asyncio.get_event_loop().run_until_complete(main())
