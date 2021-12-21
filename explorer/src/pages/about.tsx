import React from 'react';
import { Typography, Grid, Button, Steps } from 'antd'
const { Title, Paragraph } = Typography
const { Step } = Steps;
import { useIntl, FormattedMessage, IntlShape } from 'gatsby-plugin-intl';
import { OutboundLink } from "gatsby-plugin-google-gtag"
import { bodyStyles, buttonStylesLg, headingStyles, titleStyles } from '~/styles'

import { Layout } from '~/components/Layout';
import { SEO } from '~/components/SEO';

const { useBreakpoint } = Grid

import { ReactComponent as LayeredSquares } from '~/icons/layered-squares.svg';
import { ReactComponent as Hole } from '~/icons/hole.svg';
import { DOCS_URL } from '~/utils/misc/constants';

const BriefAboutSection = ({ intl, smScreen }: { intl: IntlShape, smScreen: boolean }) => (
  <div className="center-content">
    <div
      className="responsive-padding max-content-width"
      style={{
        width: '100%',
        display: 'flex',
        flexDirection: smScreen ? 'column' : 'row',
        marginBlockStart: 80,
        marginBlockEnd: 100,
      }}
    >
      <div style={{
        height: '100%',
        display: 'flex', flexDirection: 'column',
        justifyContent: 'center', zIndex: 2,
        maxWidth: 700,
      }} className="background-mask-from-left">
        <div style={{ marginBottom: smScreen ? 20 : 80, alignSelf: 'flex-start' }}>
          <Paragraph style={headingStyles} type="secondary">
            {intl.formatMessage({ id: "about.brief.heading" }).toLocaleUpperCase()}
          </Paragraph>
        </div>
        <Title level={1} style={{ ...titleStyles, fontSize: smScreen ? 44 : 56, }}>
          <FormattedMessage id="about.brief.title" />
        </Title>
        <Paragraph style={{ ...bodyStyles, marginBottom: 40, maxWidth: 560 }} type="secondary">
          <FormattedMessage id="about.brief.body" />
        </Paragraph>
      </div>
      {smScreen ? (
        <div style={{ display: 'flex', flexDirection: 'column', justifyContent: 'center', alignItems: 'center' }}>
          <LayeredSquares style={{ width: '96%' }} />
        </div>
      ) : (
        <div style={{
          display: 'flex',
          flexDirection: 'row-reverse',
          flexGrow: 1,
          minWidth: 200, maxWidth: 530,
        }}>
          <LayeredSquares style={{
            width: 520,
            minWidth: 520,
          }} />
        </div>
      )}
    </div>
  </div >
)

const HowSection = ({ intl, smScreen }: { intl: IntlShape, smScreen: boolean }) => (
  <div className="center-content blue-background">
    <div
      className="responsive-padding max-content-width"
      style={{
        marginBlock: 100,
        width: '100%',
        display: 'flex',
        flexDirection: smScreen ? 'column' : 'row',
        justifyContent: smScreen ? 'flex-start' : 'space-evenly',
        alignContent: 'center',
        alignItems: 'center'
      }}>
      <div style={{

        width: '90%',
        display: 'flex', flexDirection: 'column',
        justifyContent: 'center',

      }}>
        <Paragraph style={{ ...bodyStyles, maxWidth: smScreen ? '100%' : '80%', marginBottom: 50 }} >
          <FormattedMessage id="about.how.body" />
        </Paragraph>
        <OutboundLink
          href={DOCS_URL}
          target="_blank" rel="noopener noreferrer" className="no-external-icon"
        >
          <Button style={{ ...buttonStylesLg, marginBottom: 50 }} size="large">
            <FormattedMessage id="about.how.callToAction" />
          </Button>
        </OutboundLink>

      </div>

      <div style={{
        height: '100%',
        width: '90%',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        alignItems: 'center'
      }}>
        <Paragraph style={{ ...headingStyles, alignSelf: 'flex-start' }} type="secondary">
          {intl.formatMessage({ id: "about.how.stepsHeading" }).toLocaleUpperCase()}
        </Paragraph>
        <Steps progressDot current={6} direction="vertical">
          <Step title={intl.formatMessage({ id: 'about.how.steps.1' })} />
          <Step title={intl.formatMessage({ id: 'about.how.steps.2' })} />
          <Step title={intl.formatMessage({ id: 'about.how.steps.3' })} />
          <Step title={intl.formatMessage({ id: 'about.how.steps.4' })} />
          <Step title={intl.formatMessage({ id: 'about.how.steps.5' })} />
        </Steps>
      </div>

    </div>
  </div >
)


const ReadMoreSection = ({ smScreen }: { intl: IntlShape, smScreen: boolean }) => (
  <div className="center-content">
    <div
      className="responsive-padding max-content-width"
      style={{
        height: smScreen ? 'auto' : 800,
        width: '100%',
        position: 'relative',
        marginBlockEnd: smScreen ? 100 : 0
      }}>
      <div style={{
        position: smScreen ? 'static' : 'absolute',
        height: '100%',
        display: 'flex', flexDirection: 'column',
        justifyContent: smScreen ? 'flex-start' : 'center',
        alignItems: 'flex-start',
        zIndex: 2
      }}>
        <div style={{
          maxWidth: 500,
          marginBottom: smScreen ? 0 : 60,
          marginTop: smScreen ? 100 : 0,
        }}>
          <Paragraph
            style={{ ...headingStyles, fontSize: 24 }}
            type="secondary">
            <FormattedMessage id="about.readMore.heading" />
          </Paragraph>

        </div>
        <Title level={1} style={{
          ...titleStyles,
          fontSize: 56,
          marginTop: smScreen ? 20 : '',
          maxWidth: 800
        }}>
          <FormattedMessage id="about.readMore.title" />
        </Title>
        {/* Placeholder link to Documentation page */}
        {/* <Link to={`/${intl.locale}/documentation`}>
          <Button ghost style={{ ...buttonStylesLg, width: 255, marginTop: 30 }} size="large">
            <FormattedMessage id="about.readMore.callToAction" />
          </Button>
        </Link> */}
        {/* <OutboundLink href="mailto:contact@wormholenetwork.com" target="_blank" rel="noopener noreferrer" >
          <Button ghost style={{ ...buttonStylesLg, width: 255, marginTop: 30 }} size="large">
            <FormattedMessage id="about.emailUs" />
          </Button>
        </OutboundLink> */}
      </div>
      {smScreen ? null : (
        <div style={{ position: 'absolute', right: 0, height: '100%', display: 'flex', alignItems: 'center' }}>
          <Hole style={{ right: 0, height: '100%' }} />
        </div>
      )}
    </div>
  </div>
)

const About = () => {
  const intl = useIntl()
  const screens = useBreakpoint();

  const smScreen = screens.md === false

  return (
    <Layout>
      <SEO
        title={intl.formatMessage({ id: 'homepage.title' })}
        description={intl.formatMessage({ id: 'homepage.description' })}
      />
      <BriefAboutSection intl={intl} smScreen={smScreen} />
      <HowSection intl={intl} smScreen={smScreen} />
      <ReadMoreSection intl={intl} smScreen={smScreen} />

      <div>

      </div>
    </Layout>
  );
};

export default About
