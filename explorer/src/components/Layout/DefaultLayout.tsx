/* eslint-disable react/jsx-props-no-spreading */
import React from 'react';
import { Layout, Menu, Grid } from 'antd';
const { Header, Content, Footer } = Layout;
const { useBreakpoint } = Grid
import { MenuOutlined } from '@ant-design/icons';
import { useIntl, FormattedMessage } from 'gatsby-plugin-intl';
import { OutboundLink } from "gatsby-plugin-google-gtag"
import { useLocation } from '@reach/router';
import { Link } from 'gatsby'
import './DefaultLayout.less'

import { externalLinks, linkToService, socialLinks, socialAnchorArray } from '~/utils/misc/socials';

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
  const menuItemProps: { style: { textAlign: CanvasTextAlign, padding: number } } = { style: { textAlign: 'center', padding: 0 } }

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
                <Link to={`/${intl.locale}/network/`}>
                  <FormattedMessage id="nav.networkLink" />
                </Link>
              </Menu.Item>
            ) : null}
            {String(process.env.ENABLE_EXPLORER_PAGE) === 'true' ? (
              <Menu.Item key="explorer" {...menuItemProps}>
                <Link to={`/${intl.locale}/explorer`}>
                  <FormattedMessage id="nav.explorerLink" />
                </Link>
              </Menu.Item>
            ) : null}
            <Menu.Item key="code" {...menuItemProps}>
              <OutboundLink
                href={socialLinks['github']}
                target="_blank"
                rel="noopener noreferrer"
              >
                {intl.formatMessage({ id: "nav.codeLink" })}
              </OutboundLink>
            </Menu.Item>
            <Menu.Item key="jobs" {...menuItemProps}>
              <OutboundLink
                href={"https://boards.greenhouse.io/wormhole"}
                target="_blank"
                rel="noopener noreferrer"
              >
                {intl.formatMessage({ id: "nav.jobsLink" })}
              </OutboundLink>
            </Menu.Item>

            {screens.md === false ? (
              <Menu.Item style={{ height: '100%', padding: 0 }}>
                <Menu
                  mode="horizontal"
                  style={{ display: 'flex', justifyContent: 'space-between', width: '98vw', borderStyle: 'none' }}
                  selectedKeys={[]} >
                  {Object.entries(externalLinks).map(([url, Icon]) => <Menu.Item key={url} {...menuItemProps} style={{ margin: '12px 0' }} >
                    <div style={{ display: 'flex', justifyContent: 'space-evenly', width: '100%' }}>
                      <OutboundLink
                        href={url}
                        {...externalLinkProps}
                        title={intl.formatMessage({ id: `nav.${linkToService[url]}AltText` })}
                      >
                        <Icon style={{ height: 26 }} className="external-icon" />
                      </OutboundLink>
                    </div>
                  </Menu.Item>)}
                </Menu>
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
            <OutboundLink href={socialLinks['github']} {...externalLinkProps} style={{ color: 'white' }}>
              {intl.formatMessage({ id: "footer.openSource" })}
            </OutboundLink>
            <br />
            {intl.formatMessage({ id: "footer.createdWith" })}&nbsp;
            <span style={{ fontSize: '1.4em' }}>♥</span>
            <br />
            ©{new Date().getFullYear()}
          </div>

        </div>
      </Footer>
    </Layout>
  );
};

export default DefaultLayout;
