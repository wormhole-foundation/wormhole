import React from 'react'
import * as solanaWeb3  from '@solana/web3.js';

const ClientContext = React.createContext<solanaWeb3.Connection>(new solanaWeb3.Connection("http://localhost:8899"));
export default ClientContext
