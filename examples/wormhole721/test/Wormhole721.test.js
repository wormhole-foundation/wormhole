const {expect, assert} = require("chai")
const {deployContract} = require('./helpers')

describe("Wormhole721", function () {

  let wormhole721
  let erc721NotPlayableMock
  let playerMock

  let owner, holder

  before(async function () {
    [owner, holder] = await ethers.getSigners()
  })

  beforeEach(async function () {
    // wormhole721 = ...
  })

  it("should be implemented", async function () {
    return true
  })


})
