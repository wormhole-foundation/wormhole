const { exec } = require("child_process");

describe("Info Tests", () => {
  it("worm info contract mainnet ethereum TokenBridge", (done) => {
    exec(
      "node build/main.js info contract mainnet ethereum TokenBridge",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }

        const expectedOutput = "0x3ee18B2214AFF97000D974cf647E7C347E8fa585";

        expect(stdout.trim()).toBe(expectedOutput.trim());
        done();
      }
    );
  });

  it("worm info contract mainnet Bsc NFTBridge", (done) => {
    exec(
      "node build/main.js info contract mainnet Bsc NFTBridge",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }

        const expectedOutput = "0x5a58505a96D1dbf8dF91cB21B54419FC36e93fdE";

        expect(stdout.trim()).toBe(expectedOutput.trim());
        done();
      }
    );
  });

  it("worm info rpc mainnet Bsc", (done) => {
    exec(
      "node build/main.js info rpc mainnet Bsc",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }

        const expectedOutput = "https://bsc-dataseed.binance.org/";

        expect(stdout.trim()).toBe(expectedOutput.trim());
        done();
      }
    );
  });

  it("worm info wrapped ethereum 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48 sui", (done) => {
    exec(
      "node build/main.js info wrapped ethereum 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48 sui",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }

        const expectedOutput =
          "0x5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf::coin::COIN";

        expect(stdout.trim()).toBe(expectedOutput.trim());
        done();
      }
    );
  });

  it("worm info origin sui 0x5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf::coin::COIN", (done) => {
    exec(
      "node build/main.js info origin sui 0x5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf::coin::COIN",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          return done(new Error(`Execution error: ${error}`));
        }

        try {
          // Clean up the output to make it valid JSON
          const cleanedOutput = stdout
            .replace(/'/g, '"') // Replace single quotes with double quotes
            .replace(/(\w+):/g, '"$1":') // Add double quotes around property names
            .replace(/\x1b\[[0-9;]*m/g, "") // Remove ANSI color codes
            .trim();

          const outputObject = JSON.parse(cleanedOutput);

          const expectedOutput = {
            isWrapped: true,
            chainId: 2,
            assetAddress: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
          };

          expect(outputObject.isWrapped).toBe(expectedOutput.isWrapped);
          expect(outputObject.chainId).toBe(expectedOutput.chainId);
          expect(outputObject.assetAddress).toBe(expectedOutput.assetAddress);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm info registrations mainnet ethereum TokenBridge -v", (done) => {
    exec(
      "node build/main.js info registrations mainnet ethereum TokenBridge -v",
      (error: any, stdout: string, stderr: any) => {
        try {
          if (error) {
            done(`Execution error: ${error}`);
            return;
          }

          // Use a regular expression to extract the relevant part of stdout
          const regex = /succeeded|Mismatches/;
          const match = stdout.match(regex);
          if (!match) {
            done("The command failed to execute successfully.");
          }

          done();
        } catch (e) {
          console.log("caught a weird error", e);
          done(e);
        }
      }
    );
  });

  it("worm evm info -c Bsc -n mainnet -m TokenBridge", (done) => {
    exec(
      "node build/main.js evm info -c Bsc -n mainnet -m TokenBridge",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }

        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            address: "0xB6F6D86a8f9879A9c87f643768d9efc38c1Da6E7",
            wormhole: "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B",
            implementation: "0x621199f6beB2ba6fbD962E8A52A320EA4F6D4aA3",
            isInitialized: true,
            tokenImplementation: "0x7f8C5e730121657E17E452c5a1bA3fA1eF96f22a",
            chainId: 4,
            finality: 15,
            evmChainId: "56",
            isFork: false,
            governanceChainId: 1,
            governanceContract:
              "0x0000000000000000000000000000000000000000000000000000000000000004",
            WETH: "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c",
            registrations: {
              Solana:
                "0xec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5",
              Ethereum:
                "0x0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585",
              Terra:
                "0x0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2",
              Polygon:
                "0x0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde",
              Avalanche:
                "0x0000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052",
              Oasis:
                "0x0000000000000000000000005848c791e09901b40a9ef749f2a6735b418d7564",
              Algorand:
                "0x67e93fa6c8ac5c819990aa7340c0c16b508abb1178be9b30d024b8ac25193d45",
              Aurora:
                "0x00000000000000000000000051b5123a7b0f9b2ba265f9c4c8de7d78d52f510f",
              Fantom:
                "0x0000000000000000000000007c9fc5741288cdfdd83ceb07f3ea7e22618d79d2",
              Karura:
                "0x000000000000000000000000ae9d7fe007b3327aa64a32824aaac52c42a6e624",
              Acala:
                "0x000000000000000000000000ae9d7fe007b3327aa64a32824aaac52c42a6e624",
              Klaytn:
                "0x0000000000000000000000005b08ac39eaed75c0439fc750d9fe7e1f9dd0193f",
              Celo: "0x000000000000000000000000796dff6d74f3e27060b71255fe517bfb23c93eed",
              Near: "0x148410499d3fcda4dcfd68a1ebfcdddda16ab28326448d4aae4d2f0465cdfcb7",
              Moonbeam:
                "0x000000000000000000000000b1731c586ca89a23809861c6103f0b96b3f57d92",
              Terra2:
                "0xa463ad028fb79679cfc8ce1efba35ac0e77b35080a1abe9bebe83461f176b0a3",
              Injective:
                "0x00000000000000000000000045dbea4617971d93188eda21530bc6503d153313",
              Sui: "0xccceeb29348f71bdd22ffef43a2a19c1f5b5e17c5cca5411529120182672ade5",
              Aptos:
                "0x0000000000000000000000000000000000000000000000000000000000000001",
              Arbitrum:
                "0x0000000000000000000000000b2402144bb366a632d14b83f244d2e0e21bd39c",
              Optimism:
                "0x0000000000000000000000001d68124e65fafc907325e3edbf8c4d84499daa8b",
              Gnosis:
                "0x0000000000000000000000000000000000000000000000000000000000000000",
              Pythnet:
                "0x0000000000000000000000000000000000000000000000000000000000000000",
              Xpla: "0x8f9cf727175353b17a5f574270e370776123d90fd74956ae4277962b4fdee24c",
              Base: "0x0000000000000000000000008d2de8d2f73f1f4cab472ac9a881c9b123c79627",
              Sei: "0x86c5fd957e2db8389553e1728f9c27964b22a8154091ccba54d75f4b10c61f5e",
              Rootstock:
                "0x0000000000000000000000000000000000000000000000000000000000000000",
              Scroll:
                "0x00000000000000000000000024850c6f61c438823f01b7a3bf2b89b72174fa9d",
              Mantle:
                "0x00000000000000000000000024850c6f61c438823f01b7a3bf2b89b72174fa9d",
              Blast:
                "0x00000000000000000000000024850c6f61c438823f01b7a3bf2b89b72174fa9d",
              Xlayer:
                "0x0000000000000000000000005537857664b0f9efe38c9f320f75fef23234d904",
            },
          };

          expect(outputObject.address).toBe(expectedOutput.address);
          expect(outputObject.wormhole).toBe(expectedOutput.wormhole);
          expect(outputObject.implementation).toBe(
            expectedOutput.implementation
          );
          expect(outputObject.isInitialized).toBe(expectedOutput.isInitialized);
          expect(outputObject.tokenImplementation).toBe(
            expectedOutput.tokenImplementation
          );
          expect(outputObject.chainId).toBe(expectedOutput.chainId);
          expect(outputObject.finality).toBe(expectedOutput.finality);
          expect(outputObject.evmChainId).toBe(expectedOutput.evmChainId);
          expect(outputObject.isFork).toBe(expectedOutput.isFork);
          expect(outputObject.governanceChainId).toBe(
            expectedOutput.governanceChainId
          );
          expect(outputObject.governanceContract).toBe(
            expectedOutput.governanceContract
          );
          expect(outputObject.WETH).toBe(expectedOutput.WETH);
          expect(outputObject.registrations).toMatchObject(
            expectedOutput.registrations
          );

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });
});

describe("EVM Tests", () => {
  it("worm evm address-from-secret 0xcfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0", (done) => {
    exec(
      "node build/main.js evm address-from-secret 0xcfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }

        const expectedOutput = "0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe";

        expect(stdout.trim()).toBe(expectedOutput.trim());
        done();
      }
    );
  });
});

describe("Generate Tests", () => {
  it("worm generate registration", (done) => {
    exec(
      "node build/main.js generate registration --module NFTBridge --chain bsc --contract-address 0x706abc4E45D419950511e474C7B9Ed348A4a716c --guardian-secret cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          return done(new Error(`Execution error during generation: ${error}`));
        }

        const vaa = stdout.trim();
        expect(vaa).not.toBeNull();

        exec(
          `node build/main.js parse ${vaa}`,
          (error: any, stdout: string, stderr: any) => {
            if (error) {
              return done(new Error(`Execution error during parse: ${error}`));
            }
            try {
              const outputObject = JSON.parse(stdout);

              const expectedOutput = {
                version: 1,
                guardianSetIndex: 0,
                signatures: [
                  {
                    guardianSetIndex: 0,
                    signature:
                      "94f4939b482834dbdd5fc4391aeb42c3cedc26a24f057d01bdebfbafac33db98712c1d8ae8f4474108cd725f2371705e2de78a3ee8267d18623a6205f47b4be100",
                  },
                ],
                timestamp: 1,
                nonce: 1,
                emitterChain: 1,
                emitterAddress:
                  "0x0000000000000000000000000000000000000000000000000000000000000004",
                sequence: "8577293",
                consistencyLevel: 0,
                payload: {
                  module: "NFTBridge",
                  type: "RegisterChain",
                  chain: 0,
                  emitterChain: 4,
                  emitterAddress:
                    "0x000000000000000000000000706abc4e45d419950511e474c7b9ed348a4a716c",
                },
                digest:
                  "0x662f2eef2c8522846c34d312b3e48219d73b7d0af08f16bae95a6e4d8363c8ce",
              };

              // Can't check the signature, sequence, or digest because they are different each time.
              expect(outputObject.version).toBe(expectedOutput.version);
              expect(outputObject.guardianSetIndex).toBe(
                expectedOutput.guardianSetIndex
              );
              expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
              expect(outputObject.nonce).toBe(expectedOutput.nonce);
              expect(outputObject.emitterChain).toBe(
                expectedOutput.emitterChain
              );
              expect(outputObject.emitterAddress).toBe(
                expectedOutput.emitterAddress
              );
              expect(outputObject.consistencyLevel).toBe(
                expectedOutput.consistencyLevel
              );
              expect(outputObject.payload).toMatchObject(
                expectedOutput.payload
              );
              done();
            } catch (e) {
              done(`JSON parse error: ${e}`);
              return;
            }
          }
        );
      }
    );
  });

  it("worm generate attestation", (done) => {
    exec(
      "node build/main.js generate attestation --emitter-chain Ethereum --emitter-address 11111111111111111111111111111115 --chain Ethereum --token-address 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48 --decimals 6 --symbol USDC --name USDC --guardian-secret cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          return done(new Error(`Execution error during generation: ${error}`));
        }

        const vaa = stdout.trim();
        expect(vaa).not.toBeNull();

        exec(
          `node build/main.js parse ${vaa}`,
          (error: any, stdout: string, stderr: any) => {
            if (error) {
              return done(new Error(`Execution error during parse: ${error}`));
            }
            try {
              const outputObject = JSON.parse(stdout);

              const expectedOutput = {
                version: 1,
                guardianSetIndex: 0,
                signatures: [
                  {
                    guardianSetIndex: 0,
                    signature:
                      "342fc4c226d53b85d3ac15f88bca0585b0e4990f0d32a1ef69f1437bb0040d4b7c3d07f388b7c95646eaaaf5343e9af5fee5abc87e98b0c46334cd935d0c423200",
                  },
                ],
                timestamp: 1,
                nonce: 1,
                emitterChain: 2,
                emitterAddress:
                  "0x0000000000000000000000000000000011111111111111111111111111111115",
                sequence: "93992518",
                consistencyLevel: 0,
                payload: {
                  module: "TokenBridge",
                  chain: 0,
                  type: "AttestMeta",
                  tokenAddress:
                    "0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
                  tokenChain: 2,
                  decimals: 6,
                  symbol:
                    "USDC\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000",
                  name: "USDC\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000",
                },
                digest:
                  "0x898a01e89a1757d9cdefa63af43f7b3cce0883a2365f6c2c4103b4474e741a29",
              };

              // Can't check the signature, sequence, or digest because they are different each time.
              expect(outputObject.version).toBe(expectedOutput.version);
              expect(outputObject.guardianSetIndex).toBe(
                expectedOutput.guardianSetIndex
              );
              expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
              expect(outputObject.nonce).toBe(expectedOutput.nonce);
              expect(outputObject.emitterChain).toBe(
                expectedOutput.emitterChain
              );
              expect(outputObject.emitterAddress).toBe(
                expectedOutput.emitterAddress
              );
              expect(outputObject.consistencyLevel).toBe(
                expectedOutput.consistencyLevel
              );
              expect(outputObject.payload).toMatchObject(
                expectedOutput.payload
              );
              done();
            } catch (e) {
              done(`JSON parse error: ${e}`);
              return;
            }
          }
        );
      }
    );
  });
});

describe("Parse Tests", () => {
  it("worm parse base64-1", (done) => {
    exec(
      "node build/main.js parse AQAAAAENAnOvPC9VmJBBsAaKTq66j4glEGAhmW3mFDlUwG/Ez61cPxq6bKlRGI6WLxElHCXKmGKGheL8K2XsYJRQz8TmURIAAwDGLrDeerPQKJIPGOEN9/KgTQDriLiUCie7zozmbGdSaILJbMUN04G9v/MLtluTR8rf8JZ2cpBDr2DWUqC5BjAABPB7D+bKsYvnroJ/4RyomS/wtaKjLWW+lYIxv4TPaxT7XuuKUa3hxwqluLjPg6/jwi00cUgb2jiW6ipwRp+WkrgABXTUlnKd3m4ZCVmheUXofNleI8EAR6su71x9Dsb5EgjHJ52KGx9KYAadJZMqZ9ZV8tC0IFkAPedf08p5kv3RsNQBBlHwarb9/ULzI4QKgYs4z9HJnSI2bId5A7mN9Ava8qIELrjNDlnEY35qgKGZsRCM12WbqDcPb5R2tHmDmFTYwaYAB3hLN8YQPHUs2XpYa+jhzv8ipuSIQzKE/zHNkItcfYfiRNp1FtB6D6aSaE+Cbl5si0UgBCBtb+W65Gr7HCGM9Q0ACQtszOZ+1QHLIPsG3na5CD8TKa1404RRepSrjpqmAb56DwC7YDs2UEp03cNnNZyOoH9czVAidyzBV+APVBVjceQBC+HmtxKiNT5JB5KcQFfVur74DcCf67PcKTT0QEh5Xu+VTpkQbLKbGo2TU2na7LuLrkUZLvw87bxXMV1n7J6oAAoADdzATNdVapTotBjcOooA77Eo1PdvcUMSR6kuehmoM/wCIV0f1p4OWW2lMepYeuKsLzbSDzsZMwYK1u8+nX2EdboBDsdiFklJBq7Y2DEMMaXkpUXqKvjb447rdKPRTwc03SsaIbmFDqIObCykIkh4i/sXQ503q9ol1wW1aLJlsRO5dsUAEMYf5uvqYfLK6JXDNJZQcEh9Oatr8EQoNArw92mf3dPAHgyG2uqetElwEkiiT6TA/3X7YAssATS9cheR9mcRbkIAERm3nKolDnXaFILH9BJocwjRPvcA9ya5lBe7da5t0UP8YZ+MnCnHP5lJr/0WKcwoamGRfteFN1SkZwKSC3bPkOsAErC6teGosyHpL509+SQwGD5IQ+V8b4wVwi1isvjkuM3CdZ8oVLjRIqzbS02JKLh99BmcxoMBHS6bfUGpbYtIDMwBAAAAADnEunYAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEJWTS79YITfAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAVG9rZW5CcmlkZ2UCAAMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAJgw==",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 1,
            signatures: [
              {
                guardianSetIndex: 2,
                signature:
                  "73af3c2f55989041b0068a4eaeba8f8825106021996de6143954c06fc4cfad5c3f1aba6ca951188e962f11251c25ca98628685e2fc2b65ec609450cfc4e6511200",
              },
              {
                guardianSetIndex: 3,
                signature:
                  "00c62eb0de7ab3d028920f18e10df7f2a04d00eb88b8940a27bbce8ce66c67526882c96cc50dd381bdbff30bb65b9347cadff09676729043af60d652a0b9063000",
              },
              {
                guardianSetIndex: 4,
                signature:
                  "f07b0fe6cab18be7ae827fe11ca8992ff0b5a2a32d65be958231bf84cf6b14fb5eeb8a51ade1c70aa5b8b8cf83afe3c22d3471481bda3896ea2a70469f9692b800",
              },
              {
                guardianSetIndex: 5,
                signature:
                  "74d496729dde6e190959a17945e87cd95e23c10047ab2eef5c7d0ec6f91208c7279d8a1b1f4a60069d25932a67d655f2d0b42059003de75fd3ca7992fdd1b0d401",
              },
              {
                guardianSetIndex: 6,
                signature:
                  "51f06ab6fdfd42f323840a818b38cfd1c99d22366c877903b98df40bdaf2a2042eb8cd0e59c4637e6a80a199b1108cd7659ba8370f6f9476b479839854d8c1a600",
              },
              {
                guardianSetIndex: 7,
                signature:
                  "784b37c6103c752cd97a586be8e1ceff22a6e488433284ff31cd908b5c7d87e244da7516d07a0fa692684f826e5e6c8b452004206d6fe5bae46afb1c218cf50d00",
              },
              {
                guardianSetIndex: 9,
                signature:
                  "0b6ccce67ed501cb20fb06de76b9083f1329ad78d384517a94ab8e9aa601be7a0f00bb603b36504a74ddc367359c8ea07f5ccd5022772cc157e00f54156371e401",
              },
              {
                guardianSetIndex: 11,
                signature:
                  "e1e6b712a2353e4907929c4057d5babef80dc09febb3dc2934f44048795eef954e99106cb29b1a8d935369daecbb8bae45192efc3cedbc57315d67ec9ea8000a00",
              },
              {
                guardianSetIndex: 13,
                signature:
                  "dcc04cd7556a94e8b418dc3a8a00efb128d4f76f71431247a92e7a19a833fc02215d1fd69e0e596da531ea587ae2ac2f36d20f3b1933060ad6ef3e9d7d8475ba01",
              },
              {
                guardianSetIndex: 14,
                signature:
                  "c76216494906aed8d8310c31a5e4a545ea2af8dbe38eeb74a3d14f0734dd2b1a21b9850ea20e6c2ca42248788bfb17439d37abda25d705b568b265b113b976c500",
              },
              {
                guardianSetIndex: 16,
                signature:
                  "c61fe6ebea61f2cae895c334965070487d39ab6bf04428340af0f7699fddd3c01e0c86daea9eb449701248a24fa4c0ff75fb600b2c0134bd721791f667116e4200",
              },
              {
                guardianSetIndex: 17,
                signature:
                  "19b79caa250e75da1482c7f412687308d13ef700f726b99417bb75ae6dd143fc619f8c9c29c73f9949affd1629cc286a61917ed7853754a46702920b76cf90eb00",
              },
              {
                guardianSetIndex: 18,
                signature:
                  "b0bab5e1a8b321e92f9d3df92430183e4843e57c6f8c15c22d62b2f8e4b8cdc2759f2854b8d122acdb4b4d8928b87df4199cc683011d2e9b7d41a96d8b480ccc01",
              },
            ],
            timestamp: 0,
            nonce: 969194102,
            emitterChain: 1,
            emitterAddress:
              "0x0000000000000000000000000000000000000000000000000000000000000004",
            sequence: "2694510404604284400",
            consistencyLevel: 32,
            payload: {
              module: "TokenBridge",
              type: "ContractUpgrade",
              chain: 3,
              address:
                "0x0000000000000000000000000000000000000000000000000000000000000983",
            },
            digest:
              "0x1eb0950bc47db17fbf95bf5476a83445cf5bee90fcd14ae4d8ac851c2cbd824d",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm parse base64-2", (done) => {
    exec(
      "node build/main.js parse AQAAAAABAOKdOtGAsVPWjD9EXXXqpi/MmWkJRbqvStBGPpzkTyf3XaPUn3lyKSCqyBuivoD2iIlfF0lC/txAO8TlzjVVt3sAYrn3kQAAAAAAAgAAAAAAAAAAAAAAAPGaKgG3BRn2etswmplOyMaaln6LAAAAAAAAAAABRnJvbTogZXZtMFxuTXNnOiBIZWxsbyBXb3JsZCE=",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 0,
            signatures: [
              {
                guardianSetIndex: 0,
                signature:
                  "e29d3ad180b153d68c3f445d75eaa62fcc99690945baaf4ad0463e9ce44f27f75da3d49f79722920aac81ba2be80f688895f174942fedc403bc4e5ce3555b77b00",
              },
            ],
            timestamp: 1656354705,
            nonce: 0,
            emitterChain: 2,
            emitterAddress:
              "0x000000000000000000000000f19a2a01b70519f67adb309a994ec8c69a967e8b",
            sequence: "0",
            consistencyLevel: 1,
            payload: {
              type: "Other",
              hex: "46726f6d3a2065766d305c6e4d73673a2048656c6c6f20576f726c6421",
              ascii: "From: evm0\\nMsg: Hello World!",
            },
            digest:
              "0x8b7781f662ff1eed4827b770c0e735288948f0b56611f8cd73bf65e6b2a7a8ad",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm parse big-payload-3", (done) => {
    exec(
      "node build/main.js parse 01000000000100fd4cdd0e5a1afd9eb6555770fb132bf03ed8fa1f9e92c6adcec7881ace2ba4ba4c1b350f79da4110d3307053ceb217e4398eaf02be5474a90bd694b0d2ccbdcc0100000000baa551d500010000000000000000000000000000000000000000000000000000000000000004a3fff7bcbfc4b4ac200300000000000000000000000000000000000000000000000000000000000f4240165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa30100010000000000000000000000007c4dfd6be62406e7f5a05eec96300da4048e70ff0002000000000000000000000000000000000000000000000000000000000000000000000000000005de4c6f72656d20697073756d20646f6c6f722073697420616d65742c20636f6e73656374657475722061646970697363696e6720656c69742e204375726162697475722074656d7075732c206e6571756520656765742068656e64726572697420626962656e64756d2c20616e746520616e7465206469676e697373696d2065782c207175697320616363756d73616e20656c6974206175677565206e6563206c656f2e2050726f696e207669746165206a7573746f207669746165206c6163757320706f737565726520706f72747469746f722e204d61757269732073656420736167697474697320697073756d2e204d6f726269206d61737361206d61676e612c20706f7375657265206e6f6e20696163756c697320656765742c20756c74726963696573206174206c6967756c612e20446f6e656320756c74726963696573206e697369206573742c206574206c6f626f727469732073656d2073616769747469732073697420616d65742e20446f6e6563206665756769617420646f6c6f722061206f64696f2064696374756d2c20736564206c616f72656574206d61676e6120656765737461732e205175697371756520756c7472696369657320666163696c69736973206172637520617420616363756d73616e2e20496e20696163756c697320617420707572757320696e207472697374697175652e204d616563656e617320706f72747469746f722c206e69736c20612073656d706572206d616c6573756164612c2074656c6c7573206e65717565206d616c657375616461206c656f2c2071756973206d6f6c65737469652066656c6973206e69626820696e2065726f732e20446f6e656320766976657272612061726375206e6563206e756e63207072657469756d2c206567657420756c6c616d636f7270657220707572757320706f73756572652e2053757370656e646973736520706f74656e74692e204e616d2067726176696461206c656f206e6563207175616d2074696e636964756e7420766976657272612e205072616573656e74206163207375736369706974206f7263692e20566976616d757320736f64616c6573206d6178696d757320626c616e6469742e2050656c6c656e74657371756520696d706572646965742075726e61206174206e756e63206d616c6573756164612c20696e20617563746f72206d6173736120616c697175616d2e2050656c6c656e746573717565207363656c6572697371756520657569736d6f64206f64696f20612074656d706f722e204e756c6c612073656420706f7274612070757275732c20657520706f727461206f64696f2e20457469616d207175697320706c616365726174206e756c6c612e204e756e6320696e20636f6d6d6f646f206d692c20657520736f64616c6573206e756e632e20416c697175616d206c7563747573206c6f72656d2065742074696e636964756e74206c6163696e69612e20447569732076656c20697073756d206e69736c2e205072616573656e7420636f6e76616c6c697320656c6974206c6967756c612c206e656320706f72746120657374206d6178696d75732061632e204e756c6c61207072657469756d206c696265726f206567657420616e746520756c6c616d636f72706572206d61747469732e204e756c6c616d20766f6c75747061742c2074656c6c757320736564207363656c65726973717565206566666963697475722c206e69736c2061756775652070686172657472612066656c69732c2076656c2067726176696461206d61676e612075726e6120736564207175616d2e2044756973206964207072657469756d206475692e20496e74656765722072686f6e637573206d6174746973206a7573746f20612068656e6472657269742e20467573636520646f6c6f72206d61676e612c20706f72747469746f7220616320707572757320736f64616c65732c20657569736d6f6420766573746962756c756d20746f72746f722e20416c697175616d2070686172657472612065726174206a7573746f2c20696e20756c6c616d636f72706572207175616d2e",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 0,
            signatures: [
              {
                guardianSetIndex: 0,
                signature:
                  "fd4cdd0e5a1afd9eb6555770fb132bf03ed8fa1f9e92c6adcec7881ace2ba4ba4c1b350f79da4110d3307053ceb217e4398eaf02be5474a90bd694b0d2ccbdcc01",
              },
            ],
            timestamp: 0,
            nonce: 3131396565,
            emitterChain: 1,
            emitterAddress:
              "0x0000000000000000000000000000000000000000000000000000000000000004",
            sequence: "11817436337286722732",
            consistencyLevel: 32,
            payload: {
              module: "TokenBridge",
              type: "TransferWithPayload",
              amount: "1000000",
              tokenAddress:
                "0x165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa301",
              tokenChain: 1,
              toAddress:
                "0x0000000000000000000000007c4dfd6be62406e7f5a05eec96300da4048e70ff",
              chain: 2,
              fromAddress:
                "0x0000000000000000000000000000000000000000000000000000000000000000",
              payload:
                "0x00000000000005de4c6f72656d20697073756d20646f6c6f722073697420616d65742c20636f6e73656374657475722061646970697363696e6720656c69742e204375726162697475722074656d7075732c206e6571756520656765742068656e64726572697420626962656e64756d2c20616e746520616e7465206469676e697373696d2065782c207175697320616363756d73616e20656c6974206175677565206e6563206c656f2e2050726f696e207669746165206a7573746f207669746165206c6163757320706f737565726520706f72747469746f722e204d61757269732073656420736167697474697320697073756d2e204d6f726269206d61737361206d61676e612c20706f7375657265206e6f6e20696163756c697320656765742c20756c74726963696573206174206c6967756c612e20446f6e656320756c74726963696573206e697369206573742c206574206c6f626f727469732073656d2073616769747469732073697420616d65742e20446f6e6563206665756769617420646f6c6f722061206f64696f2064696374756d2c20736564206c616f72656574206d61676e6120656765737461732e205175697371756520756c7472696369657320666163696c69736973206172637520617420616363756d73616e2e20496e20696163756c697320617420707572757320696e207472697374697175652e204d616563656e617320706f72747469746f722c206e69736c20612073656d706572206d616c6573756164612c2074656c6c7573206e65717565206d616c657375616461206c656f2c2071756973206d6f6c65737469652066656c6973206e69626820696e2065726f732e20446f6e656320766976657272612061726375206e6563206e756e63207072657469756d2c206567657420756c6c616d636f7270657220707572757320706f73756572652e2053757370656e646973736520706f74656e74692e204e616d2067726176696461206c656f206e6563207175616d2074696e636964756e7420766976657272612e205072616573656e74206163207375736369706974206f7263692e20566976616d757320736f64616c6573206d6178696d757320626c616e6469742e2050656c6c656e74657371756520696d706572646965742075726e61206174206e756e63206d616c6573756164612c20696e20617563746f72206d6173736120616c697175616d2e2050656c6c656e746573717565207363656c6572697371756520657569736d6f64206f64696f20612074656d706f722e204e756c6c612073656420706f7274612070757275732c20657520706f727461206f64696f2e20457469616d207175697320706c616365726174206e756c6c612e204e756e6320696e20636f6d6d6f646f206d692c20657520736f64616c6573206e756e632e20416c697175616d206c7563747573206c6f72656d2065742074696e636964756e74206c6163696e69612e20447569732076656c20697073756d206e69736c2e205072616573656e7420636f6e76616c6c697320656c6974206c6967756c612c206e656320706f72746120657374206d6178696d75732061632e204e756c6c61207072657469756d206c696265726f206567657420616e746520756c6c616d636f72706572206d61747469732e204e756c6c616d20766f6c75747061742c2074656c6c757320736564207363656c65726973717565206566666963697475722c206e69736c2061756775652070686172657472612066656c69732c2076656c2067726176696461206d61676e612075726e6120736564207175616d2e2044756973206964207072657469756d206475692e20496e74656765722072686f6e637573206d6174746973206a7573746f20612068656e6472657269742e20467573636520646f6c6f72206d61676e612c20706f72747469746f7220616320707572757320736f64616c65732c20657569736d6f6420766573746962756c756d20746f72746f722e20416c697175616d2070686172657472612065726174206a7573746f2c20696e20756c6c616d636f72706572207175616d2e",
            },
            digest:
              "0xfc3ce17da88ca9085135a7180b5da44808f04fd9b55b26ed14b45cbec96a0e58",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm parse guardian-set-upgrade-1", (done) => {
    exec(
      "node build/main.js parse 010000000001007ac31b282c2aeeeb37f3385ee0de5f8e421d30b9e5ae8ba3d4375c1c77a86e77159bb697d9c456d6f8c02d22a94b1279b65b0d6a9957e7d3857423845ac758e300610ac1d2000000030001000000000000000000000000000000000000000000000000000000000000000400000000000005390000000000000000000000000000000000000000000000000000000000436f7265020000000000011358cc3ae5c097b213ce3c81979e1b9f9570746aa5ff6cb952589bde862c25ef4392132fb9d4a42157114de8460193bdf3a2fcf81f86a09765f4762fd1107a0086b32d7a0977926a205131d8731d39cbeb8c82b2fd82faed2711d59af0f2499d16e726f6b211b39756c042441be6d8650b69b54ebe715e234354ce5b4d348fb74b958e8966e2ec3dbd4958a7cdeb5f7389fa26941519f0863349c223b73a6ddee774a3bf913953d695260d88bc1aa25a4eee363ef0000ac0076727b35fbea2dac28fee5ccb0fea768eaf45ced136b9d9e24903464ae889f5c8a723fc14f93124b7c738843cbb89e864c862c38cddcccf95d2cc37a4dc036a8d232b48f62cdd4731412f4890da798f6896a3331f64b48c12d1d57fd9cbe7081171aa1be1d36cafe3867910f99c09e347899c19c38192b6e7387ccd768277c17dab1b7a5027c0b3cf178e21ad2e77ae06711549cfbb1f9c7a9d8096e85e1487f35515d02a92753504a8d75471b9f49edb6fbebc898f403e4773e95feb15e80c9a99c8348d",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 0,
            signatures: [
              {
                guardianSetIndex: 0,
                signature:
                  "7ac31b282c2aeeeb37f3385ee0de5f8e421d30b9e5ae8ba3d4375c1c77a86e77159bb697d9c456d6f8c02d22a94b1279b65b0d6a9957e7d3857423845ac758e300",
              },
            ],
            timestamp: 1628094930,
            nonce: 3,
            emitterChain: 1,
            emitterAddress:
              "0x0000000000000000000000000000000000000000000000000000000000000004",
            sequence: "1337",
            consistencyLevel: 0,
            payload: {
              module: "Core",
              type: "GuardianSetUpgrade",
              chain: 0,
              newGuardianSetIndex: 1,
              newGuardianSetLength: 19,
              newGuardianSet: [
                "58cc3ae5c097b213ce3c81979e1b9f9570746aa5",
                "ff6cb952589bde862c25ef4392132fb9d4a42157",
                "114de8460193bdf3a2fcf81f86a09765f4762fd1",
                "107a0086b32d7a0977926a205131d8731d39cbeb",
                "8c82b2fd82faed2711d59af0f2499d16e726f6b2",
                "11b39756c042441be6d8650b69b54ebe715e2343",
                "54ce5b4d348fb74b958e8966e2ec3dbd4958a7cd",
                "eb5f7389fa26941519f0863349c223b73a6ddee7",
                "74a3bf913953d695260d88bc1aa25a4eee363ef0",
                "000ac0076727b35fbea2dac28fee5ccb0fea768e",
                "af45ced136b9d9e24903464ae889f5c8a723fc14",
                "f93124b7c738843cbb89e864c862c38cddcccf95",
                "d2cc37a4dc036a8d232b48f62cdd4731412f4890",
                "da798f6896a3331f64b48c12d1d57fd9cbe70811",
                "71aa1be1d36cafe3867910f99c09e347899c19c3",
                "8192b6e7387ccd768277c17dab1b7a5027c0b3cf",
                "178e21ad2e77ae06711549cfbb1f9c7a9d8096e8",
                "5e1487f35515d02a92753504a8d75471b9f49edb",
                "6fbebc898f403e4773e95feb15e80c9a99c8348d",
              ],
            },
            digest:
              "0xed3a5600d44b9dcc889daf0178dd69ab1e9356308194ba3628a7b720ae48a8d5",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm parse nft-bridge-transfer-1", (done) => {
    exec(
      "node build/main.js parse 010000000000000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000277bb0b0001000000000000000000000000000000000000000000000000000000000000000400010000000000000000000000000000000000000000000000000000000000464f4f0000000000000000000000000000000000000000000000000000000000424152000000000000000000000000000000000000000000000000000000000000000a0a676f6f676c652e636f6d0000000000000000000000000000000000000000000000000000000000000004000a",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 0,
            signatures: [],
            timestamp: 1,
            nonce: 1,
            emitterChain: 1,
            emitterAddress:
              "0x0000000000000000000000000000000000000000000000000000000000000004",
            sequence: "41401099",
            consistencyLevel: 0,
            payload: {
              module: "NFTBridge",
              type: "Transfer",
              tokenAddress:
                "0x0000000000000000000000000000000000000000000000000000000000000004",
              tokenChain: 1,
              tokenSymbol: "FOO",
              tokenName: "BAR",
              tokenId: "10",
              tokenURI: "google.com",
              toAddress:
                "0x0000000000000000000000000000000000000000000000000000000000000004",
              chain: 10,
            },
            digest:
              "0x5e09cb958c8ee111319472907c3772f63bf4cc599b7126b1ef1bbac82f2fea7a",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm parse token-bridge-attestation-1", (done) => {
    exec(
      "node build/main.js parse 010000000001006cd3cdd701bbd878eb403f6505b5b797544eb9c486dadf79f0c445e9b8fa5cd474de1683e3a80f7e22dbfacd53b0ddc7b040ff6f974aafe7a6571c9355b8129b00000000007ce2ea3f000195f83a27e90c622a98c037353f271fd8f5f57b4dc18ebf5ff75a934724bd0491a43a1c0020f88a3e2002000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc200021200000000000000000000000000000000000000000000000000000000574554480000000000000000000000000000000000000057726170706564206574686572",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 0,
            signatures: [
              {
                guardianSetIndex: 0,
                signature:
                  "6cd3cdd701bbd878eb403f6505b5b797544eb9c486dadf79f0c445e9b8fa5cd474de1683e3a80f7e22dbfacd53b0ddc7b040ff6f974aafe7a6571c9355b8129b00",
              },
            ],
            timestamp: 0,
            nonce: 2095245887,
            emitterChain: 1,
            emitterAddress:
              "0x95f83a27e90c622a98c037353f271fd8f5f57b4dc18ebf5ff75a934724bd0491",
            sequence: "11833801757748136510",
            consistencyLevel: 32,
            payload: {
              module: "TokenBridge",
              chain: 0,
              type: "AttestMeta",
              tokenAddress:
                "0x000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
              tokenChain: 2,
              decimals: 18,
              symbol:
                "\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000WETH",
              name: "\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000Wrapped ether",
            },
            digest:
              "0x4bb52b9a44ff6062ba5db1c47afc40c186f7485c8972b1c6261eb070ce0b1c6e",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm parse token-bridge-registration-1", (done) => {
    exec(
      "node build/main.js parse 010000000001001890714264dbbc8022a58df0c12b436d588b20b6304b38c383bda1d7fc101bb2443081e6d42719bce602116a1491b10d4666967da9f8d922079759c972ed37b40100000000191428f700010000000000000000000000000000000000000000000000000000000000000004f7c884f209e7158720000000000000000000000000000000000000000000546f6b656e427269646765010000000195f83a27e90c622a98c037353f271fd8f5f57b4dc18ebf5ff75a934724bd0491",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 0,
            signatures: [
              {
                guardianSetIndex: 0,
                signature:
                  "1890714264dbbc8022a58df0c12b436d588b20b6304b38c383bda1d7fc101bb2443081e6d42719bce602116a1491b10d4666967da9f8d922079759c972ed37b401",
              },
            ],
            timestamp: 0,
            nonce: 420751607,
            emitterChain: 1,
            emitterAddress:
              "0x0000000000000000000000000000000000000000000000000000000000000004",
            sequence: "17854666897793422727",
            consistencyLevel: 32,
            payload: {
              module: "TokenBridge",
              type: "RegisterChain",
              chain: 0,
              emitterChain: 1,
              emitterAddress:
                "0x95f83a27e90c622a98c037353f271fd8f5f57b4dc18ebf5ff75a934724bd0491",
            },
            digest:
              "0xe596e88c14b9cd45c350bb4811b9a29bc1fc7069300e4204e034b1ab7c23d820",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm parse token-bridge-transfer-1", (done) => {
    exec(
      "node build/main.js parse 010000000001007d204ad9447c4dfd6be62406e7f5a05eec96300da4048e70ff530cfb52aec44807e98194990710ff166eb1b2eac942d38bc1cd6018f93662a6578d985e87c8d0016221346b0000b8bd0001c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f0000000000000003200100000000000000000000000000000000000000000000000000000002540be400165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa3010001000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d00020000000000000000000000000000000000000000000000000000000000000000",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 0,
            signatures: [
              {
                guardianSetIndex: 0,
                signature:
                  "7d204ad9447c4dfd6be62406e7f5a05eec96300da4048e70ff530cfb52aec44807e98194990710ff166eb1b2eac942d38bc1cd6018f93662a6578d985e87c8d001",
              },
            ],
            timestamp: 1646343275,
            nonce: 47293,
            emitterChain: 1,
            emitterAddress:
              "0xc69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f",
            sequence: "3",
            consistencyLevel: 32,
            payload: {
              module: "TokenBridge",
              type: "Transfer",
              amount: "10000000000",
              tokenAddress:
                "0x165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa301",
              tokenChain: 1,
              toAddress:
                "0x000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d",
              chain: 2,
              fee: "0",
            },
            digest:
              "0x2862e5873955ea104bb3e122831bdc43bbcb413da5b1123514640b950d038967",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm parse token-bridge-transfer-2", (done) => {
    exec(
      "node build/main.js parse 01000000010d0078588270e30e3b4cf74572b6ad4270cdd7932079692170fddaf369c7574722b75defcecf5d372cdd8fdba0f275c6b902434259b5d7da8402e25ca852ca5affaa0003a8888cf66158970861329efa69ff2461d847078cec22fd7f62606b17a1ae283127712fa50dc365faa1e6db339fefce57b13c74c2dce7d14b79051676c74bb685000487272398eb59763bb1e2466f9ebdea4e75c290b6c0386f07c20e1296b1976cb814547378922dbc5490b7fcf7279eafc0c08bd59ca97c4dbbcbd478967e17aa2d0006dd38ecb6233f1cd872a75cc0627ded36aa8f89095436f7dbe32e6655e27f217459fda35a3d7f1d656962160bfeee4e5fc6d2e1447559e7bc3ba760416317b86c010792d27a749b398dc5f085e7bcd2e0f18d6262a1ba1916787ec01854c0ccde0a8247f8892e6dff83fad6839fc054f32734255e9037ff9adc33499514e2300ba439010989f08688ae363783bfe3f25a5960a0791ce327bab7e7593393f91395e06fe50e3f7e13862ac86b9fd1f9720669bc4504e918f7e481c395f17a2fa131da05b9e7010a097d187970710297d188a2ebaedff0ad13efd16872566bae8a56377e28466b2c3c4e47853c60fe716109e55f8b453fb03a34bb1929c96f74ebd796a476ec7ab6000b68a19d198350b3caebd3c0159b8bbce022e0f026d013a1c83e40d6100c87e8bb0d692baca89cb77f4b6832dd7aaf3f2f7c482fd50be7221c046ae668228ec013000cd6f464a174d7e34797e2869785feb5f05ab614be989d238c9bd55259dbdbab2568c14f316d1820ac766e513bf5225185f16d30f0f01a092af5fb6b072ad577f0010d663f2f3ad62baa8ad541b9c38bb9df805d2cfa7072894526505b654293bacdee5e9e8c4ded7be92a3338b964482b3ce6d5275817d6a4b6a0663e1e84dcd1de3500105f773ea1d7e74770e78c4779abe4594b6a46f9131304948265bc185dcb1cdba8114915e3b1d864f48e4c694c9578524e22752e2d898af4b8e67383d72a11856700118bdbd5b5a820ecd215faf134b698402da04cc698e64464dd8df6692342e8c44314e1ae53bfde71fb2b00cd5691dae4f9b310c6150bdb551645a72863f4ff965c011286c673c4f2213969d273b939318f93a5b50c665efa8c9e245a3b8823522dafec209b1be127e74a6d5c924831e339f8bffb769f7b0f5772ed16231700bf7eece200624092e10000f4150001ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5000000000001aec5200100000000000000000000000000000000000000000000000000000000f4610900069b8857feab8184fb687f634618c035dac439dc1aeb3b5598a0f000000000010001000000000000000000000000efd4aa8f954ebdea82b8757c029fc8475a45e9cd00020000000000000000000000000000000000000000000000000000000000000000",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 1,
            signatures: [
              {
                guardianSetIndex: 0,
                signature:
                  "78588270e30e3b4cf74572b6ad4270cdd7932079692170fddaf369c7574722b75defcecf5d372cdd8fdba0f275c6b902434259b5d7da8402e25ca852ca5affaa00",
              },
              {
                guardianSetIndex: 3,
                signature:
                  "a8888cf66158970861329efa69ff2461d847078cec22fd7f62606b17a1ae283127712fa50dc365faa1e6db339fefce57b13c74c2dce7d14b79051676c74bb68500",
              },
              {
                guardianSetIndex: 4,
                signature:
                  "87272398eb59763bb1e2466f9ebdea4e75c290b6c0386f07c20e1296b1976cb814547378922dbc5490b7fcf7279eafc0c08bd59ca97c4dbbcbd478967e17aa2d00",
              },
              {
                guardianSetIndex: 6,
                signature:
                  "dd38ecb6233f1cd872a75cc0627ded36aa8f89095436f7dbe32e6655e27f217459fda35a3d7f1d656962160bfeee4e5fc6d2e1447559e7bc3ba760416317b86c01",
              },
              {
                guardianSetIndex: 7,
                signature:
                  "92d27a749b398dc5f085e7bcd2e0f18d6262a1ba1916787ec01854c0ccde0a8247f8892e6dff83fad6839fc054f32734255e9037ff9adc33499514e2300ba43901",
              },
              {
                guardianSetIndex: 9,
                signature:
                  "89f08688ae363783bfe3f25a5960a0791ce327bab7e7593393f91395e06fe50e3f7e13862ac86b9fd1f9720669bc4504e918f7e481c395f17a2fa131da05b9e701",
              },
              {
                guardianSetIndex: 10,
                signature:
                  "097d187970710297d188a2ebaedff0ad13efd16872566bae8a56377e28466b2c3c4e47853c60fe716109e55f8b453fb03a34bb1929c96f74ebd796a476ec7ab600",
              },
              {
                guardianSetIndex: 11,
                signature:
                  "68a19d198350b3caebd3c0159b8bbce022e0f026d013a1c83e40d6100c87e8bb0d692baca89cb77f4b6832dd7aaf3f2f7c482fd50be7221c046ae668228ec01300",
              },
              {
                guardianSetIndex: 12,
                signature:
                  "d6f464a174d7e34797e2869785feb5f05ab614be989d238c9bd55259dbdbab2568c14f316d1820ac766e513bf5225185f16d30f0f01a092af5fb6b072ad577f001",
              },
              {
                guardianSetIndex: 13,
                signature:
                  "663f2f3ad62baa8ad541b9c38bb9df805d2cfa7072894526505b654293bacdee5e9e8c4ded7be92a3338b964482b3ce6d5275817d6a4b6a0663e1e84dcd1de3500",
              },
              {
                guardianSetIndex: 16,
                signature:
                  "5f773ea1d7e74770e78c4779abe4594b6a46f9131304948265bc185dcb1cdba8114915e3b1d864f48e4c694c9578524e22752e2d898af4b8e67383d72a11856700",
              },
              {
                guardianSetIndex: 17,
                signature:
                  "8bdbd5b5a820ecd215faf134b698402da04cc698e64464dd8df6692342e8c44314e1ae53bfde71fb2b00cd5691dae4f9b310c6150bdb551645a72863f4ff965c01",
              },
              {
                guardianSetIndex: 18,
                signature:
                  "86c673c4f2213969d273b939318f93a5b50c665efa8c9e245a3b8823522dafec209b1be127e74a6d5c924831e339f8bffb769f7b0f5772ed16231700bf7eece200",
              },
            ],
            timestamp: 1648399073,
            nonce: 62485,
            emitterChain: 1,
            emitterAddress:
              "0xec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5",
            sequence: "110277",
            consistencyLevel: 32,
            payload: {
              module: "TokenBridge",
              type: "Transfer",
              amount: "4100000000",
              tokenAddress:
                "0x069b8857feab8184fb687f634618c035dac439dc1aeb3b5598a0f00000000001",
              tokenChain: 1,
              toAddress:
                "0x000000000000000000000000efd4aa8f954ebdea82b8757c029fc8475a45e9cd",
              chain: 2,
              fee: "0",
            },
            digest:
              "0xc90519b2bdfacac401d2d2c15a329d4e33e8ca15862685f0220ddc6074d7def5",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm parse token-bridge-transfer-4", (done) => {
    exec(
      "node build/main.js parse 010000000001001565b62bbf9978b1f9183ae985eb34f664fd4c850ec4ad8a38533281aec75eba2456f91c9a967cf4a70901aa0afed17ba39f1d779089b32eb88a47f7ea405e4b00000000002605c517000195f83a27e90c622a98c037353f271fd8f5f57b4dc18ebf5ff75a934724bd04914e4eb08ee374efbd200100000000000000000000000000000000000000000000000000000000000186a0165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa3010001000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c200000000000000000000000000000000000000000000000000000000000000000",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 0,
            signatures: [
              {
                guardianSetIndex: 0,
                signature:
                  "1565b62bbf9978b1f9183ae985eb34f664fd4c850ec4ad8a38533281aec75eba2456f91c9a967cf4a70901aa0afed17ba39f1d779089b32eb88a47f7ea405e4b00",
              },
            ],
            timestamp: 0,
            nonce: 637912343,
            emitterChain: 1,
            emitterAddress:
              "0x95f83a27e90c622a98c037353f271fd8f5f57b4dc18ebf5ff75a934724bd0491",
            sequence: "5642641510889746365",
            consistencyLevel: 32,
            payload: {
              module: "TokenBridge",
              type: "Transfer",
              amount: "100000",
              tokenAddress:
                "0x165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa301",
              tokenChain: 1,
              toAddress:
                "0x000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d",
              chain: 3104,
              fee: "0",
            },
            digest:
              "0x559318082f6abb8b0fcf360d2a98be84a0ccf6602044882cc0d6764a374ae44d",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm parse token-bridge-upgrade-1", (done) => {
    exec(
      "node build/main.js parse 01000000000100e3db309303b712a562e6aa2adc68bc10ff22328ab31ddb6a83706943a9da97bf11ba6e3b96395515868786898dc19ecd737d197b0d1a1f3f3c6aead5c1fe7009000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000004c5d05a00000000000000000000000000000000000000000000546f6b656e42726964676502000a0000000000000000000000000046da7a0320dd999438b4435dac82bf1dac13d2",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 0,
            signatures: [
              {
                guardianSetIndex: 0,
                signature:
                  "e3db309303b712a562e6aa2adc68bc10ff22328ab31ddb6a83706943a9da97bf11ba6e3b96395515868786898dc19ecd737d197b0d1a1f3f3c6aead5c1fe700900",
              },
            ],
            timestamp: 1,
            nonce: 1,
            emitterChain: 1,
            emitterAddress:
              "0x0000000000000000000000000000000000000000000000000000000000000004",
            sequence: "80072794",
            consistencyLevel: 0,
            payload: {
              module: "TokenBridge",
              type: "ContractUpgrade",
              chain: 10,
              address:
                "0x0000000000000000000000000046da7a0320dd999438b4435dac82bf1dac13d2",
            },
            digest:
              "0xf5725025d1d3f69d77d189e88b9be290618f1ceae355c1b116cb2d97d63f6029",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm parse token-bridge-upgrade-2", (done) => {
    exec(
      "node build/main.js parse 010000000100000000005c14b8e300010000000000000000000000000000000000000000000000000000000000000004fd81c1b1836cc25620000000000000000000000000000000000000000000546f6b656e4272696467650200030000000000000000000000000000000000000000000000000000000000000fb2",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 1,
            signatures: [],
            timestamp: 0,
            nonce: 1544861923,
            emitterChain: 1,
            emitterAddress:
              "0x0000000000000000000000000000000000000000000000000000000000000004",
            sequence: "18267094531749757526",
            consistencyLevel: 32,
            payload: {
              module: "TokenBridge",
              type: "ContractUpgrade",
              chain: 3,
              address:
                "0x0000000000000000000000000000000000000000000000000000000000000fb2",
            },
            digest:
              "0x7c8bd53e23a704a5476810d36335d2b9d65e4182e4863af3b27bd6a1ac8825e4",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm parse token-bridge-upgrade-3", (done) => {
    exec(
      "node build/main.js parse 01000000010d02aa3ffb28f2c401f90b8b5fb7af57bf9249204970053b1ba06bca147c2dab31b21d50c2a7695e8ebed00b07f0b1cb9c1658f41ee59a56b9fb4b289d9c15ce838f01030976bd5450631765db2c4017f33dd8fbd61dae364d1dfd43cdb866f72f8d4a4b47960026e64560a8823c422dce7c1e2a850aa5226c6b7092606c5224b69450e20104597391bda61d86dcb900d6b0062cf72b52b1ca250d625b3f62f89981041c16ec4b457038388058cdec9c5486baf6e84c1481e124c9f809ced7d73c235fcd400d0005c3356b43ed505d26fd55841925f8dcd4ae176990c4e9654742a20fe3438e0bb7367bd35019a546afe71af895f31e6e70de33ac800f75ee48a9dedac8f9817af5000609f9b2bd1c3f06bf23ed10eca5d13762b28872e1b44dbdb5d1f079c40c396f2a2ad2ac07278bd4cb140f61b86305f89a339dd0d39862f3a1f2cd8a9b7fd3eafe000779956931cd35e5e54934cadb37d9ad530164d58d9cbf2e39dbd73826a879705350ecd93ac7e48b21f64232b24b423d2d15b897e442dc4df9be7fdb92ebbaa2c100099af9249cfc3300edb5da61fd425371244a9a30fbfc93df455a3ac59cf3e56a4900da0566b38f8b56059f1795593b92c30c833dc6106657f4554a86f501b1f6b2010badf2b90028e98eef4aea4094c4cd0d3b411837f0469e13468b5016cc955510e808b4aafaa87aea8bc07c7506f3782d0428468df405b21fc36ed2ccfb32877642010da3f78c15d55d8efd5646a31a81c9316de35d61abd2c8a2df419b888a62a5c80771cd045db60a49d4f87c42f18de743faf6883313dba558c7da4ad209254b9b83010e5f398f609913e6d39420d44eaeff12e3431778ede50b119f0aa7ddd926de1aa34d855013984f2ca785e610e0efae0ffdb424d9caedac7f2e89ada0dee9ac6acb0110e19223db4b8837625732237d3c0f6041c28794c58178c0466ca6d646b9fb6c0941db7854a2f48a2383ea46d047e76a81f06185378c0598ead8535c0b080aa55b0111d8647c197db25ada7d55f559c65a7196fbd7361e92c68203c1842544b61bfd837117a8ab1ff10937737768c7a5fa93f65a39de2bb15370d714f1e0b4e24b6ea801127ec8835b907102f92a14a259054a1914a86afdf424108e83ae6b64b1dca0b69a6e55051cdce368fac7f3b4a50c8d9affae89d4253730eb4f12f38e056f1657ef01000000002c4929070001000000000000000000000000000000000000000000000000000000000000000411c7893f86b34cf020000000000000000000000000000000000000000000546f6b656e427269646765020004000000000000000000000000ee91c335eab126df5fdb3797ea9d6ad93aec9722",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 1,
            signatures: [
              {
                guardianSetIndex: 2,
                signature:
                  "aa3ffb28f2c401f90b8b5fb7af57bf9249204970053b1ba06bca147c2dab31b21d50c2a7695e8ebed00b07f0b1cb9c1658f41ee59a56b9fb4b289d9c15ce838f01",
              },
              {
                guardianSetIndex: 3,
                signature:
                  "0976bd5450631765db2c4017f33dd8fbd61dae364d1dfd43cdb866f72f8d4a4b47960026e64560a8823c422dce7c1e2a850aa5226c6b7092606c5224b69450e201",
              },
              {
                guardianSetIndex: 4,
                signature:
                  "597391bda61d86dcb900d6b0062cf72b52b1ca250d625b3f62f89981041c16ec4b457038388058cdec9c5486baf6e84c1481e124c9f809ced7d73c235fcd400d00",
              },
              {
                guardianSetIndex: 5,
                signature:
                  "c3356b43ed505d26fd55841925f8dcd4ae176990c4e9654742a20fe3438e0bb7367bd35019a546afe71af895f31e6e70de33ac800f75ee48a9dedac8f9817af500",
              },
              {
                guardianSetIndex: 6,
                signature:
                  "09f9b2bd1c3f06bf23ed10eca5d13762b28872e1b44dbdb5d1f079c40c396f2a2ad2ac07278bd4cb140f61b86305f89a339dd0d39862f3a1f2cd8a9b7fd3eafe00",
              },
              {
                guardianSetIndex: 7,
                signature:
                  "79956931cd35e5e54934cadb37d9ad530164d58d9cbf2e39dbd73826a879705350ecd93ac7e48b21f64232b24b423d2d15b897e442dc4df9be7fdb92ebbaa2c100",
              },
              {
                guardianSetIndex: 9,
                signature:
                  "9af9249cfc3300edb5da61fd425371244a9a30fbfc93df455a3ac59cf3e56a4900da0566b38f8b56059f1795593b92c30c833dc6106657f4554a86f501b1f6b201",
              },
              {
                guardianSetIndex: 11,
                signature:
                  "adf2b90028e98eef4aea4094c4cd0d3b411837f0469e13468b5016cc955510e808b4aafaa87aea8bc07c7506f3782d0428468df405b21fc36ed2ccfb3287764201",
              },
              {
                guardianSetIndex: 13,
                signature:
                  "a3f78c15d55d8efd5646a31a81c9316de35d61abd2c8a2df419b888a62a5c80771cd045db60a49d4f87c42f18de743faf6883313dba558c7da4ad209254b9b8301",
              },
              {
                guardianSetIndex: 14,
                signature:
                  "5f398f609913e6d39420d44eaeff12e3431778ede50b119f0aa7ddd926de1aa34d855013984f2ca785e610e0efae0ffdb424d9caedac7f2e89ada0dee9ac6acb01",
              },
              {
                guardianSetIndex: 16,
                signature:
                  "e19223db4b8837625732237d3c0f6041c28794c58178c0466ca6d646b9fb6c0941db7854a2f48a2383ea46d047e76a81f06185378c0598ead8535c0b080aa55b01",
              },
              {
                guardianSetIndex: 17,
                signature:
                  "d8647c197db25ada7d55f559c65a7196fbd7361e92c68203c1842544b61bfd837117a8ab1ff10937737768c7a5fa93f65a39de2bb15370d714f1e0b4e24b6ea801",
              },
              {
                guardianSetIndex: 18,
                signature:
                  "7ec8835b907102f92a14a259054a1914a86afdf424108e83ae6b64b1dca0b69a6e55051cdce368fac7f3b4a50c8d9affae89d4253730eb4f12f38e056f1657ef01",
              },
            ],
            timestamp: 0,
            nonce: 742992135,
            emitterChain: 1,
            emitterAddress:
              "0x0000000000000000000000000000000000000000000000000000000000000004",
            sequence: "1281143524946038000",
            consistencyLevel: 32,
            payload: {
              module: "TokenBridge",
              type: "ContractUpgrade",
              chain: 4,
              address:
                "0x000000000000000000000000ee91c335eab126df5fdb3797ea9d6ad93aec9722",
            },
            digest:
              "0x353bb7417bf9d0873091e29fc07a7b776adb780981aa217b51a6a167941c211a",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });

  it("worm parse token-bridge-upgrade-4", (done) => {
    exec(
      "node build/main.js parse 01000000010d020366c80e5ee7bed9a53f73a613072f5dedf4cbee2328ea3ab654c2ee74df1bfe0f5997aeec566ac4e5b07eea776275eb11ed74421afd18ce87501a2a4a36cbda0003385d03e9ac2739c73cbbc2a4065d3dbe103b0dd530a329b99752df31d95142ca398940a090efe7bae37f5bc5d741721b5b8fc050cedcc342904fdd4cbef6cbb20004d5ab927c8c6eb4ff77665bcfe782d761dce7d11b9d790df67f7a52ed212ca29c56f0fb7dd6fe9096f10be3339968beb60271c3956678e13711371da876908eab0105586c36c58e271b08888ed096b39ac3655e32d0b4e269b1323cf8318baf48d3ce3becaa972340b21dc5fc53287c1bf88f9cd45ed42a093a0d49d68efe824a7a90000653b21ce041d5f9a5577b65ef0fead13ce1e16abc9244296ceca2bf5a43b592dd2d2bc10d717f8c636cb3e85b6a865cf328387c009df5f600fc7ea3ac478956980007fd5c6ae5916a8002c701ce307455502aae48a8da551b919751099fb40b50f4045f22b06bcad15b9dd6a2828fc25fbcdfa6ea6b4e51a5ddeac11e1353b41bea750008558751e5db02ded0953e2b6e869d9e578d667215afb9e233d11806cfc056b83604994fb1e4ef6fea787484ad023aa9e9066052cb1a70d1ad6a7c6a8465ec4e620109bdd97efa546abbd82a8628d915af619ffd150123ba4f2cd96f9aac48c5817079474ecec60176af59b1a0bf6bce95ef973c0c5d1d2d6d2045477ff2e3031d0889000be146cbb720e243afb0d06ab1b9a732b4f09c6431beaa041c15bcac66883c8d487d294fe1c99b135cd0e6f4f964263cbee995580c5ac517708d50dba433822b59000c601a2106d69e668c9154de17edbf17eaf2a206f05d87100253c37b66f7cf58f744e42510979ddfb392964adf452a25f4c4fb86c9aadb2d02a8b54296aedb4fc8010dece66ba46b1ea485ebad2e79e6bd07fc7d4e4278edfeb79c4f5fc3eb741606ae113c0440eb40c82612390c05eddf569efab00438f19302c6c5a6769614f17c65000fb1e9f4348009f06dc34eed21fbb4f3108c4b1d9f6b6636556e95aade79979df20b6800c150dd19f0772ea393827d47f78051c1bca5e92d25834397089800012b0010f23cb11b21f103cf728b7e0d9ed4b7c1da5bbca7ca64f081bf5dc9fe1235df685c4d2265f61774b9e76f087ad115be62abbe9e3216837839c5cedab8ad0babf901000000000909bab100010000000000000000000000000000000000000000000000000000000000000004056c5c69aaf09b3420000000000000000000000000000000000000000000546f6b656e42726964676502000700000000000000000000000007b5bf20487bf1703dba0222b739fa4fc921fdd1",
      (error: any, stdout: string, stderr: any) => {
        if (error) {
          done(`Execution error: ${error}`);
          return;
        }
        try {
          const outputObject = JSON.parse(stdout);

          const expectedOutput = {
            version: 1,
            guardianSetIndex: 1,
            signatures: [
              {
                guardianSetIndex: 2,
                signature:
                  "0366c80e5ee7bed9a53f73a613072f5dedf4cbee2328ea3ab654c2ee74df1bfe0f5997aeec566ac4e5b07eea776275eb11ed74421afd18ce87501a2a4a36cbda00",
              },
              {
                guardianSetIndex: 3,
                signature:
                  "385d03e9ac2739c73cbbc2a4065d3dbe103b0dd530a329b99752df31d95142ca398940a090efe7bae37f5bc5d741721b5b8fc050cedcc342904fdd4cbef6cbb200",
              },
              {
                guardianSetIndex: 4,
                signature:
                  "d5ab927c8c6eb4ff77665bcfe782d761dce7d11b9d790df67f7a52ed212ca29c56f0fb7dd6fe9096f10be3339968beb60271c3956678e13711371da876908eab01",
              },
              {
                guardianSetIndex: 5,
                signature:
                  "586c36c58e271b08888ed096b39ac3655e32d0b4e269b1323cf8318baf48d3ce3becaa972340b21dc5fc53287c1bf88f9cd45ed42a093a0d49d68efe824a7a9000",
              },
              {
                guardianSetIndex: 6,
                signature:
                  "53b21ce041d5f9a5577b65ef0fead13ce1e16abc9244296ceca2bf5a43b592dd2d2bc10d717f8c636cb3e85b6a865cf328387c009df5f600fc7ea3ac4789569800",
              },
              {
                guardianSetIndex: 7,
                signature:
                  "fd5c6ae5916a8002c701ce307455502aae48a8da551b919751099fb40b50f4045f22b06bcad15b9dd6a2828fc25fbcdfa6ea6b4e51a5ddeac11e1353b41bea7500",
              },
              {
                guardianSetIndex: 8,
                signature:
                  "558751e5db02ded0953e2b6e869d9e578d667215afb9e233d11806cfc056b83604994fb1e4ef6fea787484ad023aa9e9066052cb1a70d1ad6a7c6a8465ec4e6201",
              },
              {
                guardianSetIndex: 9,
                signature:
                  "bdd97efa546abbd82a8628d915af619ffd150123ba4f2cd96f9aac48c5817079474ecec60176af59b1a0bf6bce95ef973c0c5d1d2d6d2045477ff2e3031d088900",
              },
              {
                guardianSetIndex: 11,
                signature:
                  "e146cbb720e243afb0d06ab1b9a732b4f09c6431beaa041c15bcac66883c8d487d294fe1c99b135cd0e6f4f964263cbee995580c5ac517708d50dba433822b5900",
              },
              {
                guardianSetIndex: 12,
                signature:
                  "601a2106d69e668c9154de17edbf17eaf2a206f05d87100253c37b66f7cf58f744e42510979ddfb392964adf452a25f4c4fb86c9aadb2d02a8b54296aedb4fc801",
              },
              {
                guardianSetIndex: 13,
                signature:
                  "ece66ba46b1ea485ebad2e79e6bd07fc7d4e4278edfeb79c4f5fc3eb741606ae113c0440eb40c82612390c05eddf569efab00438f19302c6c5a6769614f17c6500",
              },
              {
                guardianSetIndex: 15,
                signature:
                  "b1e9f4348009f06dc34eed21fbb4f3108c4b1d9f6b6636556e95aade79979df20b6800c150dd19f0772ea393827d47f78051c1bca5e92d25834397089800012b00",
              },
              {
                guardianSetIndex: 16,
                signature:
                  "f23cb11b21f103cf728b7e0d9ed4b7c1da5bbca7ca64f081bf5dc9fe1235df685c4d2265f61774b9e76f087ad115be62abbe9e3216837839c5cedab8ad0babf901",
              },
            ],
            timestamp: 0,
            nonce: 151632561,
            emitterChain: 1,
            emitterAddress:
              "0x0000000000000000000000000000000000000000000000000000000000000004",
            sequence: "390788876583607092",
            consistencyLevel: 32,
            payload: {
              module: "TokenBridge",
              type: "ContractUpgrade",
              chain: 7,
              address:
                "0x00000000000000000000000007b5bf20487bf1703dba0222b739fa4fc921fdd1",
            },
            digest:
              "0xf56a7a71e22bf768e99150e3231e201d9e7667ee28ebfb52c91260d8937b5574",
          };

          expect(outputObject.version).toBe(expectedOutput.version);
          expect(outputObject.guardianSetIndex).toBe(
            expectedOutput.guardianSetIndex
          );
          expect(outputObject.signatures).toMatchObject(
            expectedOutput.signatures
          );
          expect(outputObject.timestamp).toBe(expectedOutput.timestamp);
          expect(outputObject.nonce).toBe(expectedOutput.nonce);
          expect(outputObject.emitterChain).toBe(expectedOutput.emitterChain);
          expect(outputObject.emitterAddress).toBe(
            expectedOutput.emitterAddress
          );
          expect(outputObject.sequence).toBe(expectedOutput.sequence);
          expect(outputObject.consistencyLevel).toBe(
            expectedOutput.consistencyLevel
          );
          expect(outputObject.payload).toMatchObject(expectedOutput.payload);
          expect(outputObject.digest).toBe(expectedOutput.digest);

          done();
        } catch (e) {
          done(`JSON parse error: ${e}`);
        }
      }
    );
  });
});
