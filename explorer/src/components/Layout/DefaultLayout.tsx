/* eslint-disable react/jsx-props-no-spreading */
import React from 'react';
import { Layout, Menu } from 'antd';
const { Header, Content, Footer } = Layout;
import { useIntl, FormattedMessage } from 'gatsby-plugin-intl';
import { useLocation } from '@reach/router';
import { Link } from 'gatsby'
import './DefaultLayout.less'

type Props = {};

const DefaultLayout: React.FC<Props> = ({
  children,
  ...props
}) => {
  const intl = useIntl()
  const location = useLocation()
  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header>
          <Menu
            theme="dark"
            mode="horizontal"
            selectedKeys={[location.pathname.split('/')[2]]}
            style={{display: 'flex', justifyContent: 'space-around'}}
          >
            <Menu.Item key="">
              <Link to={`/${intl.locale}/`}>
                <span className="logo">
                  <FormattedMessage id="homepage.title" />
                </span>
              </Link>
            </Menu.Item>
            <Menu.Item key="network">
              <Link to={`/${intl.locale}/network/`}>{intl.formatMessage({id: "nav.networkLink"})}</Link>
            </Menu.Item>
            <Menu.Item key="code">
              <a
                href="https://github.com/certusone/wormhole"
                target="_blank"
                rel="noopener noreferrer"
              >
                {intl.formatMessage({id: "nav.codeLink"})}
              </a>
            </Menu.Item>
          </Menu>
      </Header>
      <Content>
        <div
          {...props}
        >
          {children}
        </div>
      </Content>
      <Footer style={{textAlign: 'center'}} className="primary-test">
        {intl.formatMessage({id: "footer.createdWith"})}{` ♥`}<br />©{new Date().getFullYear()}
      </Footer>
    </Layout>
  );
};

export default DefaultLayout;
