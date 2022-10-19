from Cryptodome.Hash import keccak
import pytest
import base64
import random
from wormhole_core import getCoreContracts
from algosdk.future import transaction
from algosdk.encoding import decode_address
from algosdk.logic import get_application_address
from algosdk.error import AlgodHTTPError
from admin import max_bits

CORE_NAME = "core"
CLEAR_NAME = "clear"
SEED_AMOUNT = 1000000
DEV_MODE = True

def pytest_namespace():
    return {'core_id': 0}

@pytest.fixture(scope='function')
def core_id():
    # Value is set after contract creation
    return pytest.core_id

@pytest.fixture(scope='function')
def boot_vaa(gen_test, portal_core, client, core_id):
    seq = int(random.random() * (2**31))
    portal_core.client = client
    portal_core.coreid = core_id
    return bytes.fromhex(gen_test.genGuardianSetUpgrade(gen_test.guardianPrivKeys, portal_core.getGovSet(), portal_core.getGovSet(), seq, seq))

@pytest.fixture(scope='function')
def core_tmpl_lsig(portal_core, boot_vaa, core_id):
    parsed_vaa = portal_core.parseVAA(boot_vaa)
    core_address = get_application_address(core_id)
    tsig = portal_core.tsig

    return tsig.populate(
            {
                "TMPL_APP_ID": core_id,
                "TMPL_APP_ADDRESS": decode_address(core_address).hex(),
                "TMPL_ADDR_IDX": int(parsed_vaa['sequence'] / max_bits),
                "TMPL_EMITTER_ID": parsed_vaa['chainRaw'].hex() + parsed_vaa['emitter'].hex(),
            }
        )

def tests_contract_creates_succesfully(gen_test, client, creator, portal_core):
    approval_program, clear_program = getCoreContracts(gen_test, CORE_NAME, CLEAR_NAME, client, SEED_AMOUNT, portal_core.tsig, DEV_MODE)
    globalSchema = transaction.StateSchema(num_uints=8, num_byte_slices=40)
    localSchema = transaction.StateSchema(num_uints=0, num_byte_slices=16)
    app_args = []

    txn = transaction.ApplicationCreateTxn(
        sender=creator.getAddress(),
        on_complete=transaction.OnComplete.NoOpOC,
        approval_program=base64.b64decode(approval_program["result"]),
        clear_program=base64.b64decode(clear_program["result"]),
        global_schema=globalSchema,
        local_schema=localSchema,
        extra_pages = 1,
        app_args=app_args,
        sp=client.suggested_params(),
    )

    signedTxn = txn.sign(creator.getPrivateKey())

    client.send_transaction(signedTxn)

    response = portal_core.waitForTransaction(client, signedTxn.get_txid())
    assert response.applicationIndex is not None and response.applicationIndex > 0
    pytest.core_id = response.applicationIndex

def tests_allow_opt_in(client, core_tmpl_lsig, creator, portal_core, suggested_params, core_id):
    core_address = get_application_address(core_id)

    seed_payment = transaction.PaymentTxn(
      sender=creator.getAddress(),
      receiver=core_tmpl_lsig.address(),
      amt=SEED_AMOUNT,
      sp=suggested_params,
    )

    seed_payment.fee = 2 * seed_payment.fee

    optin = transaction.ApplicationOptInTxn(
            sender=core_tmpl_lsig.address(),
            sp=suggested_params,
            index=core_id,
            rekey_to=core_address
            )
    optin.fee = 0

    transaction.assign_group_id([seed_payment, optin])
    signed_seed = seed_payment.sign(creator.getPrivateKey())
    signed_optin = transaction.LogicSigTransaction(optin, core_tmpl_lsig)

    client.send_transactions([signed_seed, signed_optin])
    portal_core.waitForTransaction(client, signed_optin.get_txid())

def tests_allow_init(client, creator, portal_core, suggested_params, vaa_verify_lsig, boot_vaa, core_id):
    core_address = get_application_address(core_id)

    parsed_vaa = portal_core.parseVAA(boot_vaa)
    portal_core.seed_amt = SEED_AMOUNT
    seq_addr = portal_core.optin(client, creator, core_id, int(parsed_vaa["sequence"] / max_bits), parsed_vaa["chainRaw"].hex() + parsed_vaa["emitter"].hex())
    guardian_addr = portal_core.optin(client, creator, core_id, parsed_vaa["index"], b"guardian".hex())
    newguardian_addr = portal_core.optin(client, creator, core_id, parsed_vaa["NewGuardianSetIndex"], b"guardian".hex())



    txns = [
        transaction.ApplicationCallTxn(
            sender=creator.getAddress(),
            index=core_id,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"nop", b"0"],
            sp=suggested_params
        ),

        transaction.ApplicationCallTxn(
            sender=creator.getAddress(),
            index=core_id,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"nop", b"1"],
            sp=suggested_params
        ),

        transaction.ApplicationCallTxn(
            sender=creator.getAddress(),
            index=core_id,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"init", boot_vaa, decode_address(vaa_verify_lsig.address())],
            accounts=[seq_addr, guardian_addr, newguardian_addr],
            sp=suggested_params
        ),

        transaction.PaymentTxn(
            sender=creator.getAddress(),
            receiver=vaa_verify_lsig.address(),
            amt=100000,
            sp=suggested_params
        )
    ]
    portal_core.sendTxn(client, creator, txns, True)

def tests_reject_another_init(client, creator, portal_core, suggested_params, vaa_verify_lsig,  gen_test, core_id):

    # Generate a different init vaa
    seq = int(random.random() * (2**31))
    portal_core.client = client
    portal_core.coreid = core_id
    boot_vaa = bytes.fromhex(gen_test.genGuardianSetUpgrade(gen_test.guardianPrivKeys, portal_core.getGovSet(), portal_core.getGovSet(), seq, seq))

    parsed_vaa = portal_core.parseVAA(boot_vaa)
    portal_core.seed_amt = SEED_AMOUNT
    seq_addr = portal_core.optin(client, creator, core_id, int(parsed_vaa["sequence"] / max_bits), parsed_vaa["chainRaw"].hex() + parsed_vaa["emitter"].hex())
    guardian_addr = portal_core.optin(client, creator, core_id, parsed_vaa["index"], b"guardian".hex())
    newguardian_addr = portal_core.optin(client, creator, core_id, parsed_vaa["NewGuardianSetIndex"], b"guardian".hex())



    txns = [
        transaction.ApplicationCallTxn(
            sender=creator.getAddress(),
            index=core_id,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"nop", b"0"],
            sp=suggested_params
        ),

        transaction.ApplicationCallTxn(
            sender=creator.getAddress(),
            index=core_id,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"nop", b"1"],
            sp=suggested_params
        ),

        transaction.ApplicationCallTxn(
            sender=creator.getAddress(),
            index=core_id,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"init", boot_vaa, decode_address(vaa_verify_lsig.address())],
            accounts=[seq_addr, guardian_addr, newguardian_addr],
            sp=suggested_params
        ),

        transaction.PaymentTxn(
            sender=creator.getAddress(),
            receiver=vaa_verify_lsig.address(),
            amt=100000,
            sp=suggested_params
        )
    ]

    with pytest.raises(AlgodHTTPError):
        portal_core.sendTxn(client, creator, txns, True)

def test_rejects_evil_double_verify_vaa(gen_test, portal_core, client, creator, core_id, vaa_verify_lsig):
    """
        A new verions of submitVAA. In an ideal word, we would generalize the functions
        to reduce code duplication, but at his stage I prefer duplication over complexity.
        NOTE: this reproduces an attack idea, and is not mean to reproduce a normal scenario
    """

    seq = int(random.random() * (2**31))
    signed_vaa = bytearray.fromhex(gen_test.createRandomSignedVAA(0,gen_test.guardianPrivKeys))

    trash_vaa = bytearray.fromhex(gen_test.createTrashVAA(
        guardianSetIndex=0,
        ts=1,
        nonce=1, # the nonce is irrelevant in algorand, batch not supported
        emitterChainId=8,
        emitterAddress=bytes([0x00]*32),
        sequence=seq+1,
        consistencyLevel=1,
        target="",
        payload="C0FFEEBABE",
        version=1
    ))

    # A lot of our logic here depends on parseVAA and knowing what the payload is..
    parsed_vaa = portal_core.parseVAA(signed_vaa)
    seq_addr = portal_core.optin(client, creator, core_id, int(parsed_vaa["sequence"] / max_bits), parsed_vaa["chainRaw"].hex() + parsed_vaa["emitter"].hex())

    # And then the signatures to help us verify the vaa_s
    guardian_addr = portal_core.optin(client, creator, core_id, parsed_vaa["index"], b"guardian".hex())

    accts = [seq_addr, guardian_addr]

    keys = portal_core.decodeLocalState(client, creator, core_id, guardian_addr)
    print("keys: " + keys.hex())

    sp = client.suggested_params()

    txns = []

    # How many signatures can we process in a single txn... we can do 9!
    bsize = (9*66)
    blocks = int(len(parsed_vaa["signatures"]) / bsize) + 1

    # We don't pass the entire payload in but instead just pass it pre digested.  This gets around size
    # limitations with lsigs AND reduces the cost of the entire operation on a conjested network by reducing the
    # bytes passed into the transaction
    digest = keccak.new(digest_bits=256).update(keccak.new(digest_bits=256).update(parsed_vaa["digest"]).digest()).digest()

    for i in range(blocks):
        # Which signatures will we be verifying in this block
        sigs = parsed_vaa["signatures"][(i * bsize):]
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
                sender=vaa_verify_lsig.address(),
                index=core_id,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"verifySigs", sigs, kset, digest],
                accounts=accts,
                sp=sp
            ))
        txns[-1].fee = 0

    txns.append(transaction.ApplicationCallTxn(
        sender=creator.getAddress(),
        index=core_id,
        on_complete=transaction.OnComplete.NoOpOC,
        app_args=[b"verifyVAA", signed_vaa],
        accounts=accts,
        sp=sp
    ))

    # send second, unsigned "verifyVAA" call inside
    # the transaction chain. this should obviously fail
    txns.append(transaction.ApplicationCallTxn(
        sender=creator.getAddress(),
        index=core_id,
        on_complete=transaction.OnComplete.NoOpOC,
        app_args=[b"verifyVAA", trash_vaa],
        accounts=accts,
        sp=sp
    ))
    txns[-1].fee = txns[-1].fee * (1 + blocks)

    transaction.assign_group_id(txns)

    grp = []
    pk = creator.getPrivateKey()
    for t in txns:
        if ("app_args" in t.__dict__ and len(t.app_args) > 0 and t.app_args[0] == b"verifySigs"):
            grp.append(transaction.LogicSigTransaction(t, vaa_verify_lsig))
        else:
            grp.append(t.sign(pk))

    with pytest.raises(AlgodHTTPError) as error:
        client.send_transactions(grp)
        for x in grp:
            portal_core.waitForTransaction(client, x.get_txid())

    assert "pushint 504" in str(error), f"signed_vaa:\n {signed_vaa.hex()}\n, unsigned_vaa:\n{trash_vaa.hex()}\n"
