import React from 'react'

import { IntlShape } from 'gatsby-plugin-intl';
import { OutboundLink } from "gatsby-plugin-google-gtag"
import { ReactComponent as DocsIcon } from '~/icons/Docs.svg';
import { ReactComponent as DiscordIcon } from '~/icons/Discord.svg';
import { ReactComponent as GithubIcon } from '~/icons/Github.svg';
import { ReactComponent as MediumIcon } from '~/icons/Medium.svg';
import { ReactComponent as TelegramIcon } from '~/icons/Telegram.svg';
import { ReactComponent as TwitterIcon } from '~/icons/Twitter.svg';

interface KeyToElement { [service: string]: React.SFC<React.SVGProps<SVGSVGElement>> }

const socialLinks: { [service: string]: string } = {
    'docs': "https://docs.wormholenetwork.com",
    'discord': "https://discord.gg/xsT8qrHAvV",
    'github': "https://github.com/certusone/wormhole",
    'medium': "http://wormholecrypto.medium.com",
    'telegram': "https://t.me/wormholecrypto",
    'twitter': "https://twitter.com/wormholecrypto",
}
const socialIcons: KeyToElement = {
    'docs': DocsIcon,
    'discord': DiscordIcon,
    'github': GithubIcon,
    'medium': MediumIcon,
    'telegram': TelegramIcon,
    'twitter': TwitterIcon,
}

// map of url to SVGIcon
const externalLinks = Object.keys(socialLinks).reduce<KeyToElement>((accum, service) => (
    { ...accum, [socialLinks[service]]: socialIcons[service] }
), Object())

// map of url to name
const linkToService = Object.entries(socialLinks).reduce<{ [url: string]: string }>((accum, [service, url]) => (
    { ...accum, [url]: service }
), Object())

const externalLinkProps = { target: "_blank", rel: "noopener noreferrer", className: "no-external-icon" }


const socialAnchorArray = (intl: IntlShape, linkStyles: any = {}, iconStyle: any = {}) =>
    Object.entries(externalLinks).map(([url, Icon]) => <OutboundLink
        href={url}
        key={url}
        {...externalLinkProps}
        style={linkStyles}
        title={intl.formatMessage({ id: `nav.${linkToService[url]}AltText` })}
    >
        <Icon style={iconStyle} className="external-icon" />
    </OutboundLink>)

export { socialLinks, socialIcons, externalLinks, linkToService, socialAnchorArray }
