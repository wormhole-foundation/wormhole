import React from 'react';
import { Typography, Grid, Button } from 'antd'
const { Title, Paragraph } = Typography
import { useIntl, FormattedMessage, IntlShape } from 'gatsby-plugin-intl';
import { Link } from 'gatsby'

import { Layout } from '~/components/Layout';
import { SEO } from '~/components/SEO';
import { bodyStyles, buttonStylesLg, headingStyles, titleStyles } from '~/styles'

const { useBreakpoint } = Grid

import { ReactComponent as LayeredCircles } from '~/icons/layered-circles.svg';
import { ReactComponent as CircledArrowDown } from '~/icons/circled-arrow-down.svg';
import { ReactComponent as BinanceChainIcon } from '~/icons/binancechain.svg';
import { ReactComponent as EthereumIcon } from '~/icons/ethereum.svg';
import { ReactComponent as SolanaIcon } from '~/icons/solana.svg';
import { ReactComponent as TerraIcon } from '~/icons/terra.svg';


const OpenForBizSection = ({ intl, smScreen, howAnchor }: { intl: IntlShape, smScreen: boolean, howAnchor: string }) => (

  <div className="center-content">
    <div
      className="responsive-padding max-content-width"
      style={{
        width: '100%',
        display: 'flex',
        justifyContent: 'space-around',
        marginBlock: 180,
      }}>
      <div style={{
        height: '100%',
        maxWidth: 650,
        display: 'flex', flexDirection: 'column',
        justifyContent: 'center', zIndex: 2,
        marginRight: 'auto'
      }}>
        <Title level={1} style={{ ...titleStyles, fontSize: 64 }}>
          <FormattedMessage id="homepage.openForBiz.title" />
        </Title>
        <Paragraph style={{ ...bodyStyles, marginRight: 'auto', marginBottom: 60 }} type="secondary">
          <FormattedMessage id="homepage.openForBiz.body" />
        </Paragraph>

        {/* Placeholder: call to action from designs- to explorer or elsewhere */}
        {/* <Link to={`/${intl.locale}/explorer`}>
              <Button ghost style={buttonStylesLg} size="large">
                <FormattedMessage id="homepage.openForBiz.callToAction" />
              </Button>
            </Link> */}

      </div>
      {smScreen ? null : (
        <div style={{ display: 'flex', flexDirection: 'column', placeContent: 'center', marginRight: 'auto' }}>

          {/* Placeholder: live metric of some kind from designs, commented out until we have some data to put here. */}
          {/* <div style={{ display: 'flex', flexDirection: 'column', alignContent: 'flex-end', marginTop: 80, marginRight: 80 }}>
                <Text style={{ fontSize: 16 }} type="secondary"><FormattedMessage id="homepage.openForBiz.dataLabel" /></Text>
                <Text style={{ fontSize: 24 }} type="warning">12,319,215</Text>
              </div> */}

          <Link to={'#' + howAnchor} title={intl.formatMessage({ id: "homepage.openForBiz.scrollDownAltText" })} >
            <CircledArrowDown style={{ width: 252 }} />
          </Link>


        </div>
      )}
    </div>
  </div>
)

const AboutUsSection = ({ intl, smScreen, howAnchor }: { intl: IntlShape, smScreen: boolean, howAnchor: string }) => (
  <div className="center-content blue-background">
    <div
      className="responsive-padding max-content-width"
      style={{
        width: '100%',
        display: 'flex',
        flexDirection: smScreen ? 'column' : 'row',
        justifyContent: smScreen ? 'flex-start' : 'space-between',
        marginBlockStart: smScreen ? 200 : 0,
        marginBlockEnd: smScreen ? 100 : 0,
      }}>


      {/* copy layout & formatting */}
      <div style={{
        display: 'flex',
        flexDirection: 'column',
        justifyContent: smScreen ? 'flex-start' : 'center',
        alignItems: 'flex-start',
        marginBlock: smScreen ? 0 : 200,
        zIndex: 2,
      }}>
        <div style={{ borderBottom: "0.5px solid #808088", width: 160, marginBottom: 60 }}>
          <Paragraph style={headingStyles} id={howAnchor}>
            {intl.formatMessage({ id: "homepage.aboutUs.heading" }).toLocaleUpperCase()}
          </Paragraph>
        </div>
        <Paragraph style={{ ...bodyStyles, maxWidth: smScreen ? '100%' : 500, marginBottom: 30 }} >
          <FormattedMessage id="homepage.aboutUs.body" />
        </Paragraph>
        <Link to={`/${intl.locale}/about`}>
          <Button style={buttonStylesLg} size="large">
            <FormattedMessage id="homepage.aboutUs.callToAction" />
          </Button>
        </Link>
      </div>

      {/* background image, ternary for seperate mobile layout */}
      {smScreen ? (
        <div style={{ position: 'relative', marginTop: 60, height: 260, }}>
          <div style={{ position: 'absolute', right: 40, height: '100%', display: 'flex', alignItems: 'center', zIndex: 2, }}>
            <LayeredCircles style={{ height: 260 }} />
          </div>
        </div>
      ) : (
        <div style={{
          position: 'relative', height: '100%', display: 'flex', alignItems: 'center',
        }}>
          <div style={{ position: 'absolute', right: -16, height: '100%', display: 'flex', alignItems: 'center', }}>
            <LayeredCircles style={{ height: '80%' }} />
          </div>
        </div>
      )}

    </div>
  </div>
)

const NetworkSection = ({ intl, smScreen }: { intl: IntlShape, smScreen: boolean }) => {
  const PartnersImg = <img
    src="/images/Partners.gif"
    style={{ width: 345, height: 375 }}
    alt={intl.formatMessage({ id: 'homepage.network.partnersAnimationAltText' })}
  />
  return (
    <div className="center-content">
      <div
        className="responsive-padding max-content-width"
        style={{
          width: '100%',
          minHeight: 550,
          display: 'flex',
          flexDirection: smScreen ? 'column' : 'row',
          justifyContent: smScreen ? 'center' : 'space-between',
          overflow: 'hidden',
        }}>
        <div style={{
          height: '100%',
          display: 'flex', flexDirection: 'column',
          justifyContent: smScreen ? 'flex-start' : 'center',
          paddingBlockStart: smScreen ? 100 : 0,
          zIndex: 2,
        }}>
          <div style={{ borderBottom: "0.5px solid #808088", width: 160, marginBottom: 90 }}>
            <Paragraph style={headingStyles} type="secondary">
              {intl.formatMessage({ id: "homepage.network.heading" }).toLocaleUpperCase()}
            </Paragraph>
          </div>
          <Title level={1} style={{ ...titleStyles, maxWidth: '100%', minWidth: '40%' }}>
            {intl.formatMessage({ id: "homepage.network.title" })}
          </Title>
          <Paragraph style={{ ...bodyStyles, maxWidth: '100%', minWidth: '40%', marginBottom: 30 }} type="secondary" >
            <FormattedMessage id="homepage.network.body" />
          </Paragraph>

        </div>
        {smScreen ? (
          <div style={{
            display: 'flex', justifyContent: 'center',
            alignContent: 'center', alignItems: 'center',
            marginBottom: 60, width: '100%'
          }}>
            {PartnersImg}
          </div>
        ) : (
          <div style={{
            display: 'flex', flexDirection: 'column',
            justifyContent: 'center', alignItems: 'center',
            margin: '0 8px', minWidth: '40%', maxWidth: 400
          }}>
            {PartnersImg}
          </div>
        )}
      </div>
    </div>
  )
}


const WhatWeDoSection = ({ intl, smScreen }: { intl: IntlShape, smScreen: boolean }) => {
  const iconStyles = { width: 160, margin: '0 4px' }
  const iconWithCaption = (IconEle: React.ElementType, caption: string, figureStyle: object = {}) => {
    return <figure style={{ ...iconStyles, ...figureStyle }}>
      <IconEle />
      <figcaption style={{ textAlign: 'center' }}>{caption}</figcaption>
    </figure>
  }

  return (
    <div className="center-content blue-background">
      <div
        className="responsive-padding max-content-width"
        style={{
          width: '100%',
          display: 'flex',
          flexDirection: smScreen ? 'column' : 'row',
          justifyContent: smScreen ? 'center' : 'space-between',
          marginBlock: 100,
        }}>

        <div style={{
          height: '100%',
          display: 'flex', flexDirection: 'column',
          justifyContent: smScreen ? 'flex-start' : 'center',
          // paddingBlockStart: 100,
        }}>
          <div style={{ borderBottom: "0.5px solid #808088", width: 160, marginBottom: 60 }}>
            <Paragraph style={headingStyles}>
              {intl.formatMessage({ id: "homepage.whatWeDo.heading" }).toLocaleUpperCase()}
            </Paragraph>
          </div>
          <Paragraph style={{ ...bodyStyles, maxWidth: '100%', minWidth: '40%', marginBottom: 0 }} type="secondary" >
            <FormattedMessage id="homepage.whatWeDo.problem" />
          </Paragraph>
          <Title level={1} style={{ ...titleStyles, maxWidth: '100%', minWidth: '40%', marginBottom: 40 }}>
            {intl.formatMessage({ id: "homepage.whatWeDo.title" })}
          </Title>
          <Paragraph style={{ ...bodyStyles, maxWidth: '100%', minWidth: '40%' }} type="secondary" >
            <FormattedMessage id="homepage.whatWeDo.body" />
          </Paragraph>

        </div>

        {smScreen ? (
          <div style={{
            display: 'flex', justifyContent: 'space-evenly',
            alignContent: 'flex-end', alignItems: 'baseline',
            width: '100%', height: 150
          }}>
            {iconWithCaption(EthereumIcon, 'Ethereum', { width: 70 })}
            {iconWithCaption(SolanaIcon, 'Solana', { width: 90 })}
            {iconWithCaption(TerraIcon, 'Terra', { width: 90 })}
            {iconWithCaption(BinanceChainIcon, 'Binance Smart Chain', { width: 90 })}
          </div>
        ) : (

          <div style={{
            display: 'flex', flexWrap: 'wrap',
            placeContent: 'space-evenly', alignItems: 'center',
            margin: '8px',
            maxWidth: 400,
            width: '100%',
          }}>
            {iconWithCaption(EthereumIcon, 'Ethereum', { width: 130 })}
            {iconWithCaption(SolanaIcon, 'Solana', {})}
            {iconWithCaption(TerraIcon, 'Terra', {})}
            {iconWithCaption(BinanceChainIcon, 'Binance Smart Chain', {})}
          </div>
        )}
      </div>
    </div>
  )
}

const Index = () => {
  const intl = useIntl()
  const screens = useBreakpoint();
  const smScreen = screens.md === false

  const howAnchor = intl.formatMessage({ id: "homepage.aboutUs.heading" }).replace(/\s+/g, '-').toLocaleLowerCase()

  return (
    <Layout>
      <SEO
        title={intl.formatMessage({ id: 'homepage.title' })}
        description={intl.formatMessage({ id: 'homepage.description' })}
      />
      <OpenForBizSection intl={intl} smScreen={smScreen} howAnchor={howAnchor} />
      <AboutUsSection intl={intl} smScreen={smScreen} howAnchor={howAnchor} />
      <NetworkSection intl={intl} smScreen={smScreen} />
      {/* <WhatWeDoSection intl={intl} smScreen={smScreen} /> */}

    </Layout>
  );
};

export default Index
