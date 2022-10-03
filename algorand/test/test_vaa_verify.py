import pytest
import coincurve
from algosdk.future import transaction
from algosdk.error import AlgodHTTPError
from algosdk.logic import get_application_address
from Cryptodome.Hash import keccak

@pytest.fixture(scope='module')
def app_id(portal_core, client, creator):
    return portal_core.createTestApp(client,creator)

@pytest.fixture(scope='module')
def signers(gen_test):
    return bytes.fromhex("".join(gen_test.guardianKeys))

@pytest.fixture(scope='module')
def signers_private_keys(gen_test):
    return gen_test.guardianPrivKeys

@pytest.fixture(scope='module')
def hash():
    return keccak.new(digest_bits=256).update(b"42").digest()

@pytest.fixture(scope='module')
def incorrect_hash():
    return keccak.new(digest_bits=256).update(b"error").digest()

@pytest.fixture
def signatures(gen_test,signers_private_keys, hash):
    signatures = ""
    for  i in range(len(signers_private_keys)):
        signatures += gen_test.encoder("uint8", i)

        key = coincurve.PrivateKey(bytes.fromhex(signers_private_keys[i]))
        signature = key.sign_recoverable(hash, hasher=None)
        signatures += signature.hex()
    return bytes.fromhex(signatures)

def tests_rejection_on_rekey(client, portal_core, creator, vaa_verify_lsig, app_id):
    with pytest.raises(AlgodHTTPError):
        doubleFee = client.suggested_params()
        doubleFee.flat_fee = True
        doubleFee.fee = 2000 

        feePayment = transaction.PaymentTxn(
                sender=creator.getAddress(),
                receiver=creator.getAddress(),
                amt=0,
                sp=doubleFee
                )

        zeroFee = client.suggested_params()
        zeroFee.flat_fee = True
        zeroFee.fee = 0 

        noop = transaction.ApplicationCallTxn(
                index=app_id,
                sender=vaa_verify_lsig.address(),
                sp=zeroFee,
                rekey_to=get_application_address(app_id),
                on_complete=transaction.OnComplete.NoOpOC
                )

        transaction.assign_group_id([feePayment, noop])
        signedFeePayment = feePayment.sign(creator.getPrivateKey())
        signedNoop = transaction.LogicSigTransaction(lsig=vaa_verify_lsig, transaction=noop)

        client.send_transactions([signedFeePayment, signedNoop])
        portal_core.waitForTransaction(client, signedNoop.get_txid())

def tests_rejection_on_non_app_call(client, portal_core, creator, vaa_verify_lsig):
    with pytest.raises(AlgodHTTPError):
        doubleFee = client.suggested_params()
        doubleFee.flat_fee = True
        doubleFee.fee = 2000 

        feePayment = transaction.PaymentTxn(
                sender=creator.getAddress(),
                receiver=creator.getAddress(),
                amt=0,
                sp=doubleFee
                )

        zeroFee = client.suggested_params()
        zeroFee.flat_fee = True
        zeroFee.fee = 0 

        payment = transaction.PaymentTxn(
                sender=vaa_verify_lsig.address(),
                receiver=vaa_verify_lsig.address(),
                amt=0,
                sp=zeroFee,
                )

        transaction.assign_group_id([feePayment, payment])
        signedFeePayment = feePayment.sign(creator.getPrivateKey())
        signedPayment = transaction.LogicSigTransaction(lsig=vaa_verify_lsig, transaction=payment)

        client.send_transactions([signedFeePayment, signedPayment])
        portal_core.waitForTransaction(client, signedPayment.get_txid())

def tests_rejection_on_zero_fee(client, portal_core, vaa_verify_lsig, app_id, signatures, hash, signers, suggested_params):
    with pytest.raises(AlgodHTTPError):
        noop = transaction.ApplicationCallTxn(
                index=app_id,
                sender=vaa_verify_lsig.address(),
                sp=suggested_params,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"nop", signatures, signers, hash]
                )

        signedNoop = transaction.LogicSigTransaction(lsig=vaa_verify_lsig, transaction=noop)

        client.send_transaction(signedNoop)
        portal_core.waitForTransaction(client, signedNoop.get_txid())

def tests_rejection_on_hash_not_signed(client, portal_core, creator, vaa_verify_lsig, app_id, signatures, incorrect_hash, signers):
    with pytest.raises(AlgodHTTPError):
        doubleFee = client.suggested_params()
        doubleFee.flat_fee = True
        doubleFee.fee = 2000 

        feePayment = transaction.PaymentTxn(
                sender=creator.getAddress(),
                receiver=creator.getAddress(),
                amt=0,
                sp=doubleFee
                )

        zeroFee = client.suggested_params()
        zeroFee.flat_fee = True
        zeroFee.fee = 0 

        noop = transaction.ApplicationCallTxn(
                index=app_id,
                sender=vaa_verify_lsig.address(),
                sp=zeroFee,
                on_complete=transaction.OnComplete.NoOpOC,
                app_args=[b"nop", signatures, signers, incorrect_hash]
                )

        transaction.assign_group_id([feePayment, noop])
        signedFeePayment = feePayment.sign(creator.getPrivateKey())
        signedNoop = transaction.LogicSigTransaction(lsig=vaa_verify_lsig, transaction=noop)

        client.send_transactions([signedFeePayment, signedNoop])
        portal_core.waitForTransaction(client, signedNoop.get_txid())

def tests_success(client, portal_core, creator, vaa_verify_lsig, app_id, signatures, hash, signers):
    doubleFee = client.suggested_params()
    doubleFee.flat_fee = True
    doubleFee.fee = 2000 

    feePayment = transaction.PaymentTxn(
            sender=creator.getAddress(),
            receiver=creator.getAddress(),
            amt=0,
            sp=doubleFee
            )

    zeroFee = client.suggested_params()
    zeroFee.flat_fee = True
    zeroFee.fee = 0 

    noop = transaction.ApplicationCallTxn(
            index=app_id,
            sender=vaa_verify_lsig.address(),
            sp=zeroFee,
            on_complete=transaction.OnComplete.NoOpOC,
            app_args=[b"nop", signatures, signers, hash]
            )

    transaction.assign_group_id([feePayment, noop])
    signedFeePayment = feePayment.sign(creator.getPrivateKey())
    signedNoop = transaction.LogicSigTransaction(lsig=vaa_verify_lsig, transaction=noop)

    client.send_transactions([signedFeePayment, signedNoop])
    portal_core.waitForTransaction(client, signedNoop.get_txid())

