import React from 'react'
import * as solanaWeb3  from '@solana/web3.js';
import {Account} from "@solana/web3.js";


const KeyContext = React.createContext<Account>(new Account([97,215,234,123,197,228,56,3,210,182,139,102,127,246,235,213,211,40,93,149,16,226,130,1,29,196,87,105,185,115,179,53,123,232,195,48,5,229,144,176,217,8,1,27,185,162,160,157,137,210,99,173,135,148,20,232,241,43,238,229,1,61,122,183]));
export default KeyContext
