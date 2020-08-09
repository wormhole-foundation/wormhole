import React from 'react';
import './App.css';
import * as solanaWeb3 from '@solana/web3.js';
import ClientContext from '../providers/ClientContext';
import Transfer from "../pages/Transfer";
import {Layout} from 'antd';

const {Header, Content, Footer} = Layout;

function App() {
    let c = new solanaWeb3.Connection("http://localhost:8899");
    return (
        <div className="App">
            <Layout style={{height: '100%'}}>
                <Header style={{position: 'fixed', zIndex: 1, width: '100%'}}>
                    <div className="logo"/>
                </Header>
                <Content style={{padding: '0 50px', marginTop: 64}}>
                    <div style={{padding: 24}}>
                        <ClientContext.Provider value={c}>
                            <Transfer/>
                        </ClientContext.Provider>
                    </div>
                </Content>
                <Footer style={{textAlign: 'center'}}>nexantic GmbH 2020</Footer>
            </Layout>,
        </div>
    );
}

export default App;
