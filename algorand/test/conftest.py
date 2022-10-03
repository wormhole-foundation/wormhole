import sys
sys.path.append("..")
import pytest
import base64
from admin import PortalCore
from gentest import GenTest
from algosdk.future import transaction
from vaa_verify import get_vaa_verify

@pytest.fixture(scope='module')
def portal_core():
    portal_core = PortalCore()
    portal_core.devnet = True;
    return portal_core
@pytest.fixture(scope='module')
def gen_test():
    gen_test = GenTest(False)
    return gen_test
@pytest.fixture(scope='module')
def client(portal_core): 
    return portal_core.getAlgodClient()
@pytest.fixture(scope='module')
def suggested_params(client): 
    return client.suggested_params()
@pytest.fixture(scope='module')
def creator(portal_core, client):
    return  portal_core.getTemporaryAccount(client)

@pytest.fixture(scope='module')
def vaa_verify_lsig(portal_core, client, creator, suggested_params):
    response = client.compile(get_vaa_verify())
    print(response)
    lsig = transaction.LogicSigAccount(base64.b64decode(response['result']))

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
