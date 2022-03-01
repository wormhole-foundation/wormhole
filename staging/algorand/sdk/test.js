const testConfig = require('./test-config')
const { expect } = require('chai')
const chai = require('chai')
const PricecasterSdk = require('.')
const { appId } = require('./test-config')

describe('Pricecaster SDK Tests', function () {
  let sdk
  before(async function () {
    sdk = new PricecasterSdk(testConfig.indexer.token,
      testConfig.indexer.api,
      testConfig.indexer.port,
      appId, testConfig.sourceCluster)
  })

  it('Must retrieve symbol data', async function () {
    await sdk.connect()
    expect(sdk.symbolInfo.size).to.be.greaterThan(0)
  }
  )

  it('Must query price data from contract', async function () {
    console.log(await sdk.queryData())
  })
})
