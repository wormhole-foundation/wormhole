import { LogicSigAccount } from "algosdk";
import { base64, isHexString } from "ethers/lib/utils";
import { isStringObject } from "util/types";

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
    bytecode: string;
    src: Uint8Array;
    mapByLabel: Map<TemplateName, TemplateData>;
    mapByPosition: Map<number, TemplateData>;

    constructor() {
        // Construct static template data
        const addrIdx: TemplateData = {
            name: TemplateName.TMPL_ADDR_IDX,
            bytes: false,
            position: 5,
            sourceLine: 3,
        };
        const appAddr: TemplateData = {
            name: TemplateName.TMPL_APP_ADDRESS,
            bytes: true,
            position: 90,
            sourceLine: 55,
        };
        const appId: TemplateData = {
            name: TemplateName.TMPL_APP_ID,
            bytes: false,
            position: 63,
            sourceLine: 39,
        };
        const emmId: TemplateData = {
            name: TemplateName.TMPL_EMITTER_ID,
            bytes: true,
            position: 8,
            sourceLine: 5,
        };
        const seedAmt: TemplateData = {
            name: TemplateName.TMPL_SEED_AMT,
            bytes: false,
            position: 29,
            sourceLine: 19,
        };
        // Create a template data map keyed by name
        // const tmplMapByLabel: Map<string, TemplateData> = new Map([
        this.mapByLabel = new Map([
            [TemplateName.TMPL_ADDR_IDX, addrIdx],
            [TemplateName.TMPL_APP_ADDRESS, appAddr],
            [TemplateName.TMPL_APP_ID, appId],
            [TemplateName.TMPL_EMITTER_ID, emmId],
            [TemplateName.TMPL_SEED_AMT, seedAmt],
        ]);
        // Create a template data map keyed by position
        this.mapByPosition = new Map([
            [addrIdx.position, addrIdx],
            [emmId.position, emmId],
            [seedAmt.position, seedAmt],
            [appId.position, appId],
            [appAddr.position, appAddr],
        ]);

        this.bytecode =
            "BiABAYEASIAASIgAAUMyBIEDEjMAECISEDMACIEAEhAzACAyAxIQMwAJMgMSEDMBEIEGEhAzARkiEhAzARiBABIQMwEgMgMSEDMCECISEDMCCIEAEhAzAiCAABIQMwIJMgMSEIk=";
        this.src = base64.decode(this.bytecode);
    }

    populate(data: PopulateData): LogicSigAccount {
        // populate uses the map to fill in the variable of the bytecode and returns a logic sig with the populated bytecode

        // Get the template source
        let contract: Uint8Array = this.src;
        let shift: number = 0;
        // Walk the mapByPosition and modify the bytecode in order
        this.mapByPosition.forEach((value: TemplateData, key: number) => {
            let dataVal: string | number;
            const pos: number = value.position + shift;
            switch (value.name) {
                case TemplateName.TMPL_ADDR_IDX:
                    dataVal = data.addrIdx;
                    break;
                case TemplateName.TMPL_APP_ADDRESS:
                    dataVal = data.appAddress;
                    break;
                case TemplateName.TMPL_APP_ID:
                    dataVal = data.appId;
                    break;
                case TemplateName.TMPL_EMITTER_ID:
                    dataVal = data.emitterId;
                    break;
                case TemplateName.TMPL_SEED_AMT:
                    dataVal = data.seedAmt;
                    break;
                default:
                    throw new Error("Invalid name in populate()");
            }
            if (value.bytes && isStringObject(dataVal)) {
                const val: Uint8Array = hexStringToUint8Array(dataVal);
                const lbyte: Uint8Array = hexStringToUint8Array(
                    val.length.toString(16)
                );
                // -1 to account for the existing 00 byte for length
                shift += lbyte.length - 1 + val.length;
                // +1 to overwrite the existing 00 byte for length
                const part1: Uint8Array = contract.subarray(0, pos);
                const part4: Uint8Array = contract.subarray(pos + 2);
                let len: number = 0;
                contract = new Uint8Array([]);
                contract.set(part1);
                len += part1.length;
                contract.set(lbyte, len);
                len += lbyte.length;
                contract.set(val, len);
                len += val.length;
                contract.set(part4, len);
            } else {
                const val: Uint8Array = hexStringToUint8Array(
                    dataVal.toString(16)
                );
                // -1 to account for existing 00 byte
                shift += val.length - 1;
                // +1 to overwrite existing 00 byte
                const part1: Uint8Array = contract.subarray(0, pos);
                const part4: Uint8Array = contract.subarray(pos + 2);
                let len: number = 0;
                contract = new Uint8Array([]);
                contract.set(part1);
                len += part1.length;
                contract.set(val, len);
                len += val.length;
                contract.set(part4, len);
            }
        });

        // Create a new LogicSigAccount given the populated bytecode
        return new LogicSigAccount(contract);
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
