import algosdk, { LogicSigAccount } from "algosdk";
import { base64, id, isHexString } from "ethers/lib/utils";
import { isStringObject } from "util/types";
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
    return Uint8Array.from(Buffer.from(hs, "hex"));
}

export class TmplSig {
    algoClient: algosdk.Algodv2
    sourceHash: string
    bytecode: Uint8Array

    constructor(algoClient: algosdk.Algodv2) {
        this.algoClient = algoClient
        this.sourceHash = ""
        this.bytecode = new Uint8Array
    }
    
    async compile(source: string) {
        const hash = id(source)
        if (hash !== this.sourceHash) {
            const response = await this.algoClient.compile(source).do()
            this.bytecode = new Uint8Array(Buffer.from(response.result, 'base64'))
            this.sourceHash = hash
        }
    }

    /**
     * Populate data in the TEAL source and return the LogicSig object based on the resulting compiled bytecode.
     * @param data The data to populate fields with. 
     * @notes emitterId must be prefixed with '0x'. appAddress must be decoded with algoSDK and prefixed with '0x'.
     * @returns A LogicSig object.
     */
    async populate(data: PopulateData): Promise<LogicSigAccount> {
        let program = tealSource
        program = program.replace(/TMPL_ADDR_IDX/, data.addrIdx.toString())
        program = program.replace(/TMPL_EMITTER_ID/, data.emitterId)
        program = program.replace(/TMPL_SEED_AMT/, data.seedAmt.toString())
        program = program.replace(/TMPL_APP_ID/, data.appId.toString())
        program = program.replace(/TMPL_APP_ADDRESS/, data.appAddress)
        await this.compile(program)

        // Create a new LogicSigAccount given the populated TEAL code
        return new LogicSigAccount(this.bytecode);
    }

    //     def get_bytecode_chunk(self, idx: int) -> Bytes:
    //         start = 0
    //         if idx > 0:
    //             start = list(self.sorted.values())[idx - 1]["position"] + 1

    //         stop = len(self.src)
    //         if idx < len(self.sorted):
    //             stop = list(self.sorted.values())[idx]["position"]

    //         chunk = self.src[start:stop]
    //         return Bytes(chunk)

    //     def get_sig_tmpl(self):
    //         def sig_tmpl():
    //             # We encode the app id as an 8 byte integer to ensure its a known size
    //             # Otherwise the uvarint encoding may produce a different byte offset
    //             # for the template variables
    //             admin_app_id = Tmpl.Int("TMPL_APP_ID")
    //             admin_address = Tmpl.Bytes("TMPL_APP_ADDRESS")
    //             seed_amt = Tmpl.Int("TMPL_SEED_AMT")

    //             @Subroutine(TealType.uint64)
    //             def init():
    //                 algo_seed = Gtxn[0]
    //                 optin = Gtxn[1]
    //                 rekey = Gtxn[2]

    //                 return And(
    //                     Global.group_size() == Int(3),

    //                     algo_seed.type_enum() == TxnType.Payment,
    //                     algo_seed.amount() == seed_amt,
    //                     algo_seed.rekey_to() == Global.zero_address(),
    //                     algo_seed.close_remainder_to() == Global.zero_address(),

    //                     optin.type_enum() == TxnType.ApplicationCall,
    //                     optin.on_completion() == OnComplete.OptIn,
    //                     optin.application_id() == admin_app_id,
    //                     optin.rekey_to() == Global.zero_address(),

    //                     rekey.type_enum() == TxnType.Payment,
    //                     rekey.amount() == Int(0),
    //                     rekey.rekey_to() == admin_address,
    //                     rekey.close_remainder_to() == Global.zero_address(),
    //                 )

    //             return Seq(
    //                 # Just putting adding this as a tmpl var to make the address unique and deterministic
    //                 # We don't actually care what the value is, pop it
    //                 Pop(Tmpl.Int("TMPL_ADDR_IDX")),
    //                 Pop(Tmpl.Bytes("TMPL_EMITTER_ID")),
    //                 init(),
    //             )

    //         return compileTeal(sig_tmpl(), mode=Mode.Signature, version=6, assembleConstants=True)

    // if __name__ == '__main__':
    //     core = TmplSig("sig")
    // #    client =  AlgodClient("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "http://localhost:4001")
    // #    pprint.pprint(client.compile( core.get_sig_tmpl()))

    //     with open("sig.tmpl.teal", "w") as f:
    //         f.write(core.get_sig_tmpl())
}
