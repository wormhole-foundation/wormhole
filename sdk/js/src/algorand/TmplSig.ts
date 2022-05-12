import algosdk, { LogicSigAccount } from "algosdk";
import { id } from "ethers/lib/utils";
import { hexToUint8Array } from "../utils";
import { encodeHex } from "./BigVarint";

// This is the data structure to be populated in the call to populate() below
// Yes, it needs to be filled out before calling populate()
interface IPopulateData {
  appId: bigint;
  appAddress: string;
  addrIdx: bigint;
  emitterId: string;
}
export type PopulateData = Required<IPopulateData>;

export class TmplSig {
  algoClient: algosdk.Algodv2;
  sourceHash: string;
  bytecode: Uint8Array;

  constructor(algoClient: algosdk.Algodv2) {
    this.algoClient = algoClient;
    this.sourceHash = "";
    this.bytecode = new Uint8Array();
  }

  async compile(source: string) {
    const hash = id(source);
    if (hash !== this.sourceHash) {
      const response = await this.algoClient.compile(source).do();
      this.bytecode = new Uint8Array(Buffer.from(response.result, "base64"));
      this.sourceHash = hash;
    }
  }

  /**
   * Populate data in the TEAL source and return the LogicSig object based on the resulting compiled bytecode.
   * @param data The data to populate fields with.
   * @notes emitterId must be prefixed with '0x'. appAddress must be decoded with algoSDK and prefixed with '0x'.
   * @returns A LogicSig object.
   */

  async populate(data: PopulateData): Promise<LogicSigAccount> {
    const byteString: string = [
      "0620010181",
      encodeHex(data.addrIdx),
      "4880",
      encodeHex(BigInt(data.emitterId.length / 2)),
      data.emitterId,
      "483110810612443119221244311881",
      encodeHex(data.appId),
      "1244312080",
      encodeHex(BigInt(data.appAddress.length / 2)),
      data.appAddress,
      "124431018100124431093203124431153203124422",
    ].join("");
    this.bytecode = hexToUint8Array(byteString);
    return new LogicSigAccount(this.bytecode);
  }
}
