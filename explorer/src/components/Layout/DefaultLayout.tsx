/* eslint-disable react/jsx-props-no-spreading */
import React from 'react';
import { Layout, Menu, Grid } from 'antd';
const { Header, Content, Footer } = Layout;
const { useBreakpoint } = Grid
import { MenuOutlined } from '@ant-design/icons';
import { useIntl, FormattedMessage } from 'gatsby-plugin-intl';
import { useLocation } from '@reach/router';
import { Link } from 'gatsby'
import './DefaultLayout.less'

import { socialLinks, socialAnchorArray } from '~/utils/misc/socials';

// brand assets
import { ReactComponent as AvatarAndName } from '~/icons/FullLogo_DarkBackground.svg';
import { ReactComponent as Avatar } from '~/icons/Avatar_DarkBackground.svg';

const externalLinkProps = { target: "_blank", rel: "noopener noreferrer", className: "no-external-icon" }

const DefaultLayout: React.FC<{}> = ({
  children,
  ...props
}) => {
  const intl = useIntl()
  const location = useLocation()
  const screens = useBreakpoint();
  const menuItemProps: { style: { textAlign: CanvasTextAlign } } = { style: { textAlign: 'center' } }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{
        padding: 0,
        height: 70
      }} >
        <div className="center-content">
          <Menu
            mode="horizontal"
            selectedKeys={[location.pathname.split('/')[2]]}
            style={{
              height: 70,
              display: 'flex',
              width: '100%',
              padding: !screens.md ? 0 : '0 80px 0 148px'
            }}
            overflowedIndicator={<MenuOutlined style={{ fontSize: '24px', verticalAlign: 'middle', marginRight: 0 }} />}
            className="max-content-width"
          >
            <Menu.Item key="" className="responsive-padding" >
              <Link to={`/${intl.locale}/`} style={{ height: 32 }} title={intl.formatMessage({ id: 'nav.homeLinkAltText' })}>
                <AvatarAndName style={{ height: 45, margin: 'auto', verticalAlign: 'middle', display: 'inline-block' }} />
              </Link>
            </Menu.Item>
            <div style={{ flexGrow: 1, minWidth: '20%' }}>
              {/* pushes the elements away on both sides */}
            </div>
            <Menu.Item key="about" {...menuItemProps}>
              <Link to={`/${intl.locale}/about`}>
                <FormattedMessage id="nav.aboutLink" />
              </Link>
            </Menu.Item>
            {String(process.env.ENABLE_NETWORK_PAGE) === 'true' ? (
              <Menu.Item key="network" {...menuItemProps}>
                <Link to={`/${intl.locale}/network/`}>{intl.formatMessage({ id: "nav.networkLink" })}</Link>
              </Menu.Item>
            ) : null}
            <Menu.Item key="code" {...menuItemProps}>
              <a
                href={socialLinks['github']}
                target="_blank"
                rel="noopener noreferrer"
              >
                {intl.formatMessage({ id: "nav.codeLink" })}
              </a>
            </Menu.Item>

            {screens.md === false ? (
              <Menu.Item key="external" style={{ margin: '12px 0' }}>
                <div style={{ display: 'flex', justifyContent: 'space-evenly', width: '100vw' }}>
                  {socialAnchorArray(intl, { zIndex: 2 }, { height: 26 })}
                </div>
              </Menu.Item>
            ) : null}
          </Menu>
        </div>
        <div
          className="external-links-left"
        >
          {socialAnchorArray(intl)}
        </div>
      </Header>

      <Content>
        <div
          {...props}
        >
          {children}
        </div>
      </Content>

      <Footer style={{ textAlign: 'center', paddingLeft: 0, paddingRight: 0 }}>
        <div
          className="external-links-bottom"
        >
          {socialAnchorArray(intl)}
        </div>
        <div style={{
          display: 'flex',
          justifyContent: 'center',
          alignContent: 'center',
          alignItems: 'center',
          gap: 16,
          marginTop: 12
        }}>
          <Avatar style={{ maxHeight: 58 }} />
          <div style={{ lineHeight: '1.5em' }}>
            <a href={socialLinks['github']} {...externalLinkProps} style={{ color: 'white' }}>
              {intl.formatMessage({ id: "footer.openSource" })}
            </a>
            <br />
            {intl.formatMessage({ id: "footer.createdWith" })}&nbsp;
            <a href="https://certus.one/" {...externalLinkProps} style={{ color: 'white' }}>
              <span style={{ fontSize: '1.4em' }}>♥</span>
            </a>
            <br />
            ©{new Date().getFullYear()}
          </div>

        </div>
      </Footer>
    </Layout>
  );
};

export default DefaultLayout;
