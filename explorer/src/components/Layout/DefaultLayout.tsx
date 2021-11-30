/* eslint-disable react/jsx-props-no-spreading */
import React, { useEffect, useState } from 'react';
import { Button, Layout, Grid } from 'antd';
const { Header, Content, Footer } = Layout;
const { useBreakpoint } = Grid
import { SendOutlined } from '@ant-design/icons';
import { useIntl, FormattedMessage } from 'gatsby-plugin-intl';
import { OutboundLink } from "gatsby-plugin-google-gtag"
import { Link } from 'gatsby'
import './DefaultLayout.less'
import styled from "styled-components";


import { externalLinks, linkToService, socialLinks, socialAnchorArray } from '~/utils/misc/socials';

// brand assets
import { ReactComponent as AvatarAndName } from '~/icons/FullLogo_DarkBackground.svg';
import { ReactComponent as Avatar } from '~/icons/Avatar_DarkBackground.svg';
import { BRIDGE_URL, DOCS_URL, JOBS_URL } from '~/utils/misc/constants';


const Toggle = styled.div`
    display: none;
    height: 100%;
    cursor: pointer;
    padding: 0 4vw;
    @media (max-width: 992px) {
        display: flex;
    }
`;

const Navbox = styled.div`
    align-items: center;
    @media (max-width: 992px) {
        flex-direction: column;
        position: fixed;
        width: 100%;
        justify-content: flex-start;
        padding-top: 360px;
        background-color: #010114;
        transition: all 0.3s ease-in;
        left: ${(props: { open: boolean }) => (props.open ? "-100%" : "0")};
    }
`;

const Hamburger = styled.div`
    background-color: #fff;
    width: 30px;
    height: 3px;
    transition: all 0.3s linear;
    align-self: center;
    position: relative;
    transform: ${(props: { open: boolean }) => (props.open ? "rotate(-45deg)" : "inherit")};
    z-index: 1001;
    ::before,
    ::after {
        width: 30px;
        height: 3px;
        background-color: #fff;
        content: "";
        position: absolute;
        transition: all 0.3s linear;
    }
    ::before {
        transform: ${(props) =>
    props.open
      ? "rotate(-90deg) translate(-10px, 0px)"
      : "rotate(0deg)"};
        top: -10px;
    }
    ::after {
        opacity: ${(props) => (props.open ? "0" : "1")};
        transform: ${(props) =>
    props.open ? "rotate(90deg) " : "rotate(0deg)"};
        top: 10px;
    }
`;

const externalLinkProps = { target: "_blank", rel: "noopener noreferrer", className: "no-external-icon" }

const DefaultLayout: React.FC<{}> = ({
  children,
  ...props
}) => {
  const intl = useIntl()
  const screens = useBreakpoint();
  const [navbarOpen, setNavbarOpen] = useState(false);
  const menuItemProps: { style: { textAlign: CanvasTextAlign, padding: number } } = { style: { textAlign: 'center', padding: 0 } }

  useEffect(() => {
    if (screens.lg === true) {
      setNavbarOpen(false)
    }
  }, [screens.lg])

  const launchBridge = <div key="bridge" style={{ ...menuItemProps.style, zIndex: 1001 }}>
    <OutboundLink
      href={BRIDGE_URL}
      target="_blank"
      rel="noopener noreferrer"
      className="no-external-icon"
    >
      <Button
        style={{
          height: 40,
          fontSize: 16,
          border: "1.5px solid",
          paddingLeft: 20
        }}
        ghost
        type="primary"
        shape="round"
        size="large"
      >
        {intl.formatMessage({ id: "nav.bridgeLink" })}
        <SendOutlined style={{ fontSize: 16, marginRight: 0 }} />
      </Button>
    </OutboundLink>
  </div>

  const menuItems = [
    <div key="about" {...menuItemProps}>
      <Link to={`/${intl.locale}/about/`}>
        <FormattedMessage id="nav.aboutLink" />
      </Link>
    </div>,
    <div key="network" {...menuItemProps} >
      <Link to={`/${intl.locale}/network/`}>
        <FormattedMessage id="nav.networkLink" />
      </Link>
    </div>,
    <div key="explorer" {...menuItemProps} >
      <Link to={`/${intl.locale}/explorer/`}>
        <FormattedMessage id="nav.explorerLink" />
      </Link>
    </div>,
    <div key="docs" {...menuItemProps} >
      <OutboundLink
        href={DOCS_URL}
        target="_blank"
        rel="noopener noreferrer"
      >
        {intl.formatMessage({ id: "nav.docsLink" })}
      </OutboundLink>
    </div>,
    <div key="jobs" {...menuItemProps} >
      <OutboundLink
        href={JOBS_URL}
        target="_blank"
        rel="noopener noreferrer"
      >
        {intl.formatMessage({ id: "nav.jobsLink" })}
      </OutboundLink>
    </div>,
    screens.sm === false || screens.lg === true ? launchBridge : null,
    screens.lg === false ? (<div key="socials" style={{ ...menuItemProps.style, height: '100%', padding: 0 }}>
      <div
        style={{ display: 'flex', justifyContent: 'space-evenly', borderStyle: 'none' }}
      >
        {Object.entries(externalLinks).map(([url, Icon]) => <div key={url} {...menuItemProps} style={{ margin: '12px 0' }} >
          <OutboundLink
            href={url}
            {...externalLinkProps}
            title={intl.formatMessage({ id: `nav.${linkToService[url]}AltText` })}
          >
            <Icon style={{ height: 26 }} className="external-icon" />
          </OutboundLink>
        </div>)}
      </div>
    </div>) : null
  ]
  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{
        padding: 0,
        height: 70
      }} >
        <div className="center-content">
          <nav
            className={`max-content-width ${navbarOpen ? " affix" : ""}`}
            style={{
              height: 70,
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              width: '100%',
              padding: !screens.lg ? 0 : '0 16px 0 0'
            }}
          >
            {/* wormhole logo, left side of nav */}
            <div className="responsive-padding" style={{ zIndex: 1001 }}>
              <Link to={`/${intl.locale}/`} style={{ height: 32 }} title={intl.formatMessage({ id: 'nav.homeLinkAltText' })}>
                <AvatarAndName style={{ height: 45, margin: 'auto', verticalAlign: 'middle', display: 'inline-block' }} />
              </Link>
            </div>

            {/* the list of menu items, right side of nav */}
            <div className="nav site-nav-right">
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 16 }} >
                {menuItems}
              </div>
            </div>

            {/* show the "Launch Bridge" button next to the hamburger menu if the screen is large enough. */}
            {screens.lg === false && screens.sm === true ? <>
              <div style={{ flexGrow: 1 }} />
              {launchBridge}
            </> : null}

            {/* hambuger button Toggle mobile popover menu*/}
            <Toggle onClick={() => setNavbarOpen(!navbarOpen)}>
              {navbarOpen ? <Hamburger open /> : <Hamburger open={false} />}
            </Toggle>

            {/* nav drawer with links */}
            {navbarOpen ? (
              <Navbox open={!navbarOpen}>
                <div className="popover" style={{ marginTop: 100 }}>
                  {/* <Navigation data={navigation} /> */}
                  <div className="nav" style={{ display: 'flex' }}>
                    {menuItems}
                  </div>
                </div>
              </Navbox>
            ) : null}

          </nav>
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
