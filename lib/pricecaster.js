const algosdk = require('algosdk')
const fs = require('fs')
const tools = require('../tools/app-tools')

const approvalProgramFilename = 'teal/pricekeeper.teal'
const clearProgramFilename = 'teal/clearstate.teal'

class PricecasterLib {
  constructor (algodClient, ownerAddr = undefined) {
    this.algodClient = algodClient
    this.ownerAddr = ownerAddr
    this.minFee = 1000

    /**
         * Set Application Id used in all the functions of this class.
         * @param {number} applicationId application id
         * @returns {void}
         */
    this.setAppId = function (applicationId) {
      this.appId = applicationId
    }

    /**
         * Get minimum fee to pay for transactions.
         * @return {Number} minimum transaction fee
         */
    this.minTransactionFee = function () {
      return this.minFee
    }

    /**
         * Internal function.
         * Read application local state related to the account.
         * @param  {String} accountAddr account to retrieve local state
         * @return {Array} an array containing all the {key: value} pairs of the local state
         */
    this.readLocalState = function (accountAddr) {
      return tools.readAppLocalState(this.algodClient, this.appId, accountAddr)
    }

    /**
         * Internal function.
         * Read application global state.
         * @return {Array} an array containing all the {key: value} pairs of the global state
         * @returns {void}
         */
    this.readGlobalState = function () {
      return tools.readAppGlobalState(this.algodClient, this.appId, this.ownerAddr)
    }

    /**
         * Print local state of accountAddr on stdout.
         * @param  {String} accountAddr account to retrieve local state
         * @returns {void}
         */
    this.printLocalState = async function (accountAddr) {
      await tools.printAppLocalState(this.algodClient, this.appId, accountAddr)
    }

    /**
         * Print application global state on stdout.
         * @returns {void}
         */
    this.printGlobalState = async function () {
      await tools.printAppGlobalState(this.algodClient, this.appId, this.ownerAddr)
    }

    /**
         * Internal function.
         * Read application local state variable related to accountAddr.
         * @param  {String} accountAddr account to retrieve local state
         * @param  {String} key variable key to get the value associated
         * @return {String/Number} it returns the value associated to the key that could be an address, a number or a
         * base64 string containing a ByteArray
         */
    this.readLocalStateByKey = function (accountAddr, key) {
      return tools.readAppLocalStateByKey(this.algodClient, this.appId, accountAddr, key)
    }

    /**
         * Internal function.
         * Read application global state variable.
         * @param  {String} key variable key to get the value associated
         * @return {String/Number} it returns the value associated to the key that could be an address,
         * a number or a base64 string containing a ByteArray
         */
    this.readGlobalStateByKey = function (key) {
      return tools.readAppGlobalStateByKey(this.algodClient, this.appId, this.ownerAddr, key)
    }

    /**
         * Compile program that programFilename contains.
         * @param  {String} programFilename filepath to the program to compile
         * @return {String} base64 string containing the compiled program
         */
    this.compileProgram = async function (programBytes) {
      const compileResponse = await this.algodClient.compile(programBytes).do()
      const compiledBytes = new Uint8Array(Buffer.from(compileResponse.result, 'base64'))
      return compiledBytes
    }

    /**
         * Internal function.
         * Compile application clear state program.
         * @return {String} base64 string containing the compiled program
         */
    this.compileClearProgram = function () {
      const program = fs.readFileSync(clearProgramFilename, 'utf8')
      return this.compileProgram(program)
    }

    /**
         * Internal function.
         * Compile application approval program.
         * @return {String} base64 string containing the compiled program
         */
    this.compileApprovalProgram = async function (validatorAddress) {
      const program = fs.readFileSync(approvalProgramFilename, 'utf8')
      return this.compileProgram(program)
    }

    /**
         * Helper function to retrieve the application id from a createApp transaction response.
         * @param  {Object} txResponse object containig the transactionResponse of the createApp call
         * @return {Number} application id of the created application
         */
    this.appIdFromCreateAppResponse = function (txResponse) {
      return txResponse['application-index']
    }

    /**
         * Create an application based on the default approval and clearState programs or based on the specified files.
         * @param  {String} sender account used to sign the createApp transaction
         * @param  {Function} signCallback callback with prototype signCallback(sender, tx) used to sign transactions
         * @return {String} transaction id of the created application
         */
    this.createApp = async function (sender, validatorAddr, symbol, signCallback) {
      const localInts = 0
      const localBytes = 0
      const globalInts = 2
      const globalBytes = 4

      // declare onComplete as NoOp
      const onComplete = algosdk.OnApplicationComplete.NoOpOC

      // get node suggested parameters
      const params = await algodClient.getTransactionParams().do()

      params.fee = this.minFee
      params.flatFee = true

      const approvalProgramCompiled = await this.compileApprovalProgram()
      const clearProgramCompiled = await this.compileClearProgram()

      const enc = new TextEncoder()
      const appArgs = [new Uint8Array(algosdk.decodeAddress(validatorAddr).publicKey), enc.encode(symbol)]

      // create unsigned transaction
      const txApp = algosdk.makeApplicationCreateTxn(
        sender, params, onComplete,
        approvalProgramCompiled, clearProgramCompiled,
        localInts, localBytes, globalInts, globalBytes, appArgs
      )
      const txId = txApp.txID().toString()

      // Sign the transaction
      const txAppSigned = signCallback(sender, txApp)

      // Submit the transaction
      await algodClient.sendRawTransaction(txAppSigned).do()
      return txId
    }

    /**
         * Internal function.
         * Call application specifying args and accounts.
         * @param  {String} sender caller address
         * @param  {Array} appArgs array of arguments to pass to application call
         * @param  {Array} appAccounts array of accounts to pass to application call
         * @param  {Function} signCallback callback with prototype signCallback(sender, tx) used to sign transactions
         * @return {String} transaction id of the transaction
         */
    this.callApp = async function (sender, appArgs, appAccounts, signCallback) {
      // get node suggested parameters
      const params = await this.algodClient.getTransactionParams().do()

      params.fee = this.minFee
      params.flatFee = true

      // create unsigned transaction
      const txApp = algosdk.makeApplicationNoOpTxn(sender, params, this.appId, appArgs, appAccounts.length === 0 ? undefined : appAccounts)
      const txId = txApp.txID().toString()

      // Sign the transaction
      const txAppSigned = signCallback(sender, txApp)

      // Submit the transaction
      await this.algodClient.sendRawTransaction(txAppSigned).do()

      return txId
    }

    /**
         * ClearState sender. Remove all the sender associated local data.
         * @param  {String} sender account to ClearState
         * @param  {Function} signCallback callback with prototype signCallback(sender, tx) used to sign transactions
         * @return {[String]} transaction id of one of the transactions of the group
         */
    this.clearApp = async function (sender, signCallback, forcedAppId) {
      // get node suggested parameters
      const params = await this.algodClient.getTransactionParams().do()

      params.fee = this.minFee
      params.flatFee = true

      let appId = this.appId
      if (forcedAppId) {
        appId = forcedAppId
      }

      // create unsigned transaction
      const txApp = algosdk.makeApplicationClearStateTxn(sender, params, appId)
      const txId = txApp.txID().toString()

      // Sign the transaction
      const txAppSigned = signCallback(sender, txApp)

      // Submit the transaction
      await this.algodClient.sendRawTransaction(txAppSigned).do()

      return txId
    }

    /**
        * Permanent delete the application.
        * @param  {String} sender owner account
        * @param  {Function} signCallback callback with prototype signCallback(sender, tx) used to sign transactions
        * @param  {Function} applicationId use this application id instead of the one set
        * @return {String}      transaction id of one of the transactions of the group
        */
    this.deleteApp = async function (sender, signCallback, applicationId) {
      // get node suggested parameters
      const params = await this.algodClient.getTransactionParams().do()

      params.fee = this.minFee
      params.flatFee = true

      if (!applicationId) {
        applicationId = this.appId
      }

      // create unsigned transaction
      const txApp = algosdk.makeApplicationDeleteTxn(sender, params, applicationId)
      const txId = txApp.txID().toString()

      // Sign the transaction
      const txAppSigned = signCallback(sender, txApp)

      // Submit the transaction
      await this.algodClient.sendRawTransaction(txAppSigned).do()

      return txId
    }

    /**
         * Helper function to wait until transaction txId is included in a block/round.
         * @param  {String} txId transaction id to wait for
         * @return {VOID} VOID
         */
    this.waitForConfirmation = async function (txId) {
      const status = (await this.algodClient.status().do())
      let lastRound = status['last-round']
      // eslint-disable-next-line no-constant-condition
      while (true) {
        const pendingInfo = await this.algodClient.pendingTransactionInformation(txId).do()
        if (pendingInfo['confirmed-round'] !== null && pendingInfo['confirmed-round'] > 0) {
          // Got the completed Transaction

          return pendingInfo['confirmed-round']
        }
        lastRound += 1
        await this.algodClient.statusAfterBlock(lastRound).do()
      }
    }

    /**
         * Helper function to wait until transaction txId is included in a block/round
         * and returns the transaction response associated to the transaction.
         * @param  {String} txId transaction id to get transaction response
         * @return {Object}      returns an object containing response information
         */
    this.waitForTransactionResponse = async function (txId) {
      // Wait for confirmation
      await this.waitForConfirmation(txId)

      // display results
      return this.algodClient.pendingTransactionInformation(txId).do()
    }

    /**
     * Creates a message with price data for the PriceKeeper contract
     * @param {BigInt} nonce Sequence number
     * @param {String} symbol Symbol, must match appid support, 16-char UTF long
     * @param {number} price Price, expected in 64-bit floating-point format.
     * @param {number} confidence Confidence, expected in 64-bit floating-point format.
     * @param {Uint8Array} sk Signing key.
     * @returns A base64-encoded message.
     */
    this.createMessage = function (nonce, symbol, price, confidence, sk) {
      const buf = Buffer.alloc(131)
      buf.write('PRICEDATA', 0)
      buf.writeInt8(1, 9)
      buf.writeBigUInt64BE(BigInt(this.appId), 10)
      buf.writeBigUInt64BE(nonce, 18)
      buf.write(symbol, 26)
      buf.writeDoubleBE(price, 42)
      buf.writeDoubleBE(confidence, 50)
      buf.writeBigUInt64BE(BigInt(Math.floor(Date.now() / 1000)), 58)

      const signature = Buffer.from(algosdk.signBytes(buf, sk))
      signature.copy(buf, 66)

      // v-component (ignored in Algorand it seems)
      buf.writeInt8(1, 130)
      return buf.toString('base64')
    }

    /**
     * Submits message to the PriceKeeper contract.
     * @param {*} sender Sender account
     * @param {*} msgb64 Base64-encoded message.
     * @returns Transaction identifier (txid)
     */
    this.submitMessage = async function (sender, msgb64, signCallback) {
      return await this.callApp(sender, [new Uint8Array(msgb64)], [], signCallback)
    }
  }
}

module.exports = {
  PricecasterLib
}
