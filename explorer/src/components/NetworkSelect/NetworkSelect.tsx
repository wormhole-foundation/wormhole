import React from 'react';
import { Select } from 'antd'
const { Option } = Select
import { FormattedMessage } from 'gatsby-plugin-intl'
import { NetworkContext } from "./network-context"

const NetworkSelect = ({ style }: { style?: { [key: string]: string | number } }) => {
    return (
        <NetworkContext.Consumer>
            {({ activeNetwork, setActiveNetwork }) => (
                <Select
                    defaultValue={activeNetwork.name}
                    onSelect={setActiveNetwork}
                    size="large"
                    style={style}
                >
                    <Option value="devnet"><FormattedMessage id="networks.devnet" /></Option>
                    <Option value="testnet"><FormattedMessage id="networks.testnet" /></Option>
                    <Option value="mainnet"><FormattedMessage id="networks.mainnet" /></Option>
                </Select>
            )}
        </NetworkContext.Consumer>

    )

}

export default NetworkSelect
