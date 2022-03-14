import algosdk, { LogicSigAccount } from "algosdk";
import { id } from "ethers/lib/utils";
import { tealSource } from "./TmplSigSource";

enum TemplateName {
    TMPL_ADDR_IDX = 0,
    TMPL_APP_ADDRESS,
    TMPL_APP_ID,
    TMPL_EMITTER_ID,
    TMPL_SEED_AMT,
}

// This is an entry in the template data table
interface ITemplateData {
    name: TemplateName;
    bytes: boolean;
    position: number;
    sourceLine: number;
}
export type TemplateData = Required<ITemplateData>;

// This is the data structure to be populated in the call to populate() below
// Yes, it needs to be filled out before calling populate()
interface IPopulateData {
    seedAmt: number;
    appId: number;
    appAddress: string;
    addrIdx: number;
    emitterId: string;
}
export type PopulateData = Required<IPopulateData>;

// Maybe move this to a helpers file
export function hexStringToUint8Array(hs: string): Uint8Array {
    if (hs.length % 2 === 1) {
        // prepend a 0
        hs = "0" + hs;
    }
    const buf = Buffer.from(hs, "hex");
    const retval = Uint8Array.from(buf);
    console.log("input:", hs, ", buf:", buf, ", retval:", retval);
    return retval;
    // return Uint8Array.from(Buffer.from(hs, "hex"));
}

export function uint8ArrayToHexString(arr: Uint8Array, add0x: boolean) {
    const ret: string = Buffer.from(arr).toString("hex");
    if (!add0x) {
        return ret;
    }
    return "0x" + ret;
}

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
            this.bytecode = new Uint8Array(
                Buffer.from(response.result, "base64")
            );
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
        let program = tealSource;
        program = program.replace(/TMPL_ADDR_IDX/, data.addrIdx.toString());
        program = program.replace(/TMPL_EMITTER_ID/, data.emitterId);
        program = program.replace(/TMPL_SEED_AMT/, data.seedAmt.toString());
        program = program.replace(/TMPL_APP_ID/, data.appId.toString());
        program = program.replace(/TMPL_APP_ADDRESS/, data.appAddress);
        await this.compile(program);

        console.log(
            "This is the final product:",
            Buffer.from(this.bytecode).toString("hex")
        );
        // Create a new LogicSigAccount given the populated TEAL code
        return new LogicSigAccount(this.bytecode);
    }
}
