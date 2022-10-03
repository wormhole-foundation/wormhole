from algosdk.encoding import decode_address
import pytest
from algosdk.future import transaction
from algosdk.logic import get_application_address
from algosdk.error import AlgodHTTPError

@pytest.fixture(scope='module')
def correct_app_id(portal_core, client, creator):
    return portal_core.createTestApp(client,creator)
@pytest.fixture(scope='module')
def incorrect_app_id(portal_core, client, creator):
    return portal_core.createTestApp(client,creator)
@pytest.fixture(scope='module')
def tmpl_lsig(portal_core, client, correct_app_id, creator, suggested_params):
    appAddress = get_application_address(correct_app_id)
    tsig = portal_core.tsig

    lsig = tsig.populate(
            {
                "TMPL_APP_ID": correct_app_id,
                "TMPL_APP_ADDRESS": decode_address(appAddress).hex(),
                "TMPL_ADDR_IDX": 0,
                "TMPL_EMITTER_ID": b"emitter".hex(),
            }
        )

    txn = transaction.PaymentTxn(
      sender=creator.getAddress(),
      receiver=lsig.address(),
      amt=1000000,
      sp=suggested_params,
    )

    signedTxn = txn.sign(creator.getPrivateKey())

    client.send_transaction(signedTxn)
    portal_core.waitForTransaction(client, signedTxn.get_txid())
    return lsig

def tests_rejection_on_payment(client, portal_core, tmpl_lsig, creator, suggested_params):
    with pytest.raises(AlgodHTTPError):
        feePayment = transaction.PaymentTxn(
                sender=creator.getAddress(),
                receiver=creator.getAddress(),
                amt=0,
                sp=suggested_params
                )
        feePayment.fee = 2 * feePayment.fee

        payment = transaction.PaymentTxn(
                sender=tmpl_lsig.address(),
                receiver=tmpl_lsig.address(),
                amt=0,
                sp=suggested_params
                )

        payment.fee = 0

        transaction.assign_group_id([feePayment, payment])
        signedFeePayment = feePayment.sign(creator.getPrivateKey())
        signedPayment = transaction.LogicSigTransaction(lsig=tmpl_lsig, transaction=payment)

        client.send_transactions([signedFeePayment, signedPayment])
        portal_core.waitForTransaction(client, signedPayment.get_txid())

def tests_rejection_on_asset_transfer(client, portal_core, tmpl_lsig, creator, suggested_params):
    with pytest.raises(AlgodHTTPError):
        fee_payment = transaction.PaymentTxn(
                sender=creator.getAddress(),
                receiver=creator.getAddress(),
                amt=0,
                sp=suggested_params
                )

        fee_payment.fee = 2 * fee_payment.fee

        asset_transfer = transaction.AssetTransferTxn(
                index=1,
                sender=tmpl_lsig.address(),
                receiver=tmpl_lsig.address(),
                amt=0,
                sp=suggested_params
                )

        asset_transfer.fee = 0

        transaction.assign_group_id([fee_payment, asset_transfer])
        signedFeePayment = fee_payment.sign(creator.getPrivateKey())
        signedAssetTransfer = transaction.LogicSigTransaction(lsig=tmpl_lsig, transaction=asset_transfer)

        client.send_transactions([signedFeePayment, signedAssetTransfer])
        portal_core.waitForTransaction(client, signedAssetTransfer.get_txid())

def tests_rejection_on_nop(client, portal_core, tmpl_lsig, correct_app_id, creator, suggested_params):
    with pytest.raises(AlgodHTTPError):
        fee_payment = transaction.PaymentTxn(
                sender=creator.getAddress(),
                receiver=creator.getAddress(),
                amt=0,
                sp=suggested_params
                )

        fee_payment.fee = 2 * fee_payment.fee


        noop = transaction.ApplicationCallTxn(
                index=correct_app_id,
                sender=tmpl_lsig.address(),
                sp=suggested_params,
                on_complete=transaction.OnComplete.NoOpOC
                )
        noop.fee = 0

        transaction.assign_group_id([fee_payment, noop])
        signedFeePayment = fee_payment.sign(creator.getPrivateKey())
        signedNoop = transaction.LogicSigTransaction(lsig=tmpl_lsig, transaction=noop)

        client.send_transactions([signedFeePayment, signedNoop])
        portal_core.waitForTransaction(client, signedNoop.get_txid())

def tests_rejection_on_opt_in_to_incorrect_app(client, portal_core, tmpl_lsig, incorrect_app_id, creator, suggested_params):
    with pytest.raises(AlgodHTTPError):
        fee_payment = transaction.PaymentTxn(
                sender=creator.getAddress(),
                receiver=creator.getAddress(),
                amt=0,
                sp=suggested_params
                )

        fee_payment.fee = 2*fee_payment.fee 

        opt_in = transaction.ApplicationCallTxn(
                index=incorrect_app_id,
                sender=tmpl_lsig.address(),
                sp=suggested_params,
                on_complete=transaction.OnComplete.OptInOC
                )

        opt_in.fee = 0

        transaction.assign_group_id([fee_payment, opt_in])
        signedFeePayment = fee_payment.sign(creator.getPrivateKey())
        signedOptIn = transaction.LogicSigTransaction(lsig=tmpl_lsig, transaction=opt_in)

        client.send_transactions([signedFeePayment, signedOptIn])
        portal_core.waitForTransaction(client, signedOptIn.get_txid())

def tests_rejection_on_opt_in_to_correct_app_without_rekeying(client, portal_core, tmpl_lsig, correct_app_id, creator, suggested_params):
    with pytest.raises(AlgodHTTPError):
        fee_payment = transaction.PaymentTxn(
                sender=creator.getAddress(),
                receiver=creator.getAddress(),
                amt=0,
                sp=suggested_params
                )

        fee_payment.fee = 2*fee_payment.fee 

        opt_in = transaction.ApplicationCallTxn(
                index=correct_app_id,
                sender=tmpl_lsig.address(),
                sp=suggested_params,
                on_complete=transaction.OnComplete.OptInOC
                )

        opt_in.fee = 0

        transaction.assign_group_id([fee_payment, opt_in])
        signedFeePayment = fee_payment.sign(creator.getPrivateKey())
        signedOptIn = transaction.LogicSigTransaction(lsig=tmpl_lsig, transaction=opt_in)

        client.send_transactions([signedFeePayment, signedOptIn])
        portal_core.waitForTransaction(client, signedOptIn.get_txid())

def tests_rejection_on_opt_in_to_correct_app_with_rekeying_with_non_zero_fee(client, portal_core, tmpl_lsig, correct_app_id,  suggested_params):
    with pytest.raises(AlgodHTTPError):
        txn = transaction.ApplicationCallTxn(
                index=correct_app_id,
                sender=tmpl_lsig.address(),
                sp=suggested_params,
                rekey_to=get_application_address(correct_app_id),
                on_complete=transaction.OnComplete.NoOpOC
                )
        signedTxn = transaction.LogicSigTransaction(lsig=tmpl_lsig, transaction=txn)
        client.send_transaction(signedTxn)
        portal_core.waitForTransaction(client, signedTxn.get_txid())

def tests_success(client, portal_core, creator, tmpl_lsig, correct_app_id, suggested_params):
    fee_payment = transaction.PaymentTxn(
            sender=creator.getAddress(),
            receiver=creator.getAddress(),
            amt=0,
            sp=suggested_params
            )

    fee_payment.fee = 2*fee_payment.fee 

    opt_in = transaction.ApplicationCallTxn(
            index=correct_app_id,
            sender=tmpl_lsig.address(),
            sp=suggested_params,
            rekey_to=get_application_address(correct_app_id),
            on_complete=transaction.OnComplete.OptInOC
            )

    opt_in.fee = 0

    transaction.assign_group_id([fee_payment, opt_in])
    signedFeePayment = fee_payment.sign(creator.getPrivateKey())
    signedOptIn = transaction.LogicSigTransaction(lsig=tmpl_lsig, transaction=opt_in)

    client.send_transactions([signedFeePayment, signedOptIn])
    portal_core.waitForTransaction(client, signedOptIn.get_txid())
