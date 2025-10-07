import { Config } from '@ton/blueprint';
import { ScaffoldPlugin } from 'blueprint-scaffold';

export const config: Config = {
    plugins: [new ScaffoldPlugin()],
};