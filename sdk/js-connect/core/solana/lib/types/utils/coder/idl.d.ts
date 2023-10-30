import { Layout } from 'buffer-layout';
import { anchor } from '@wormhole-foundation/connect-sdk-solana';
export declare class IdlCoder {
    static fieldLayout(field: {
        name?: string;
    } & Pick<anchor.IdlField, 'type'>, types?: anchor.IdlTypeDef[]): Layout;
}
