import React from 'react'
import * as solanaWeb3  from '@solana/web3.js';
import {SOLANA_HOST} from "../config";

const ClientContext = React.createContext<solanaWeb3.Connection>(new solanaWeb3.Connection(SOLANA_HOST));
export default ClientContext
