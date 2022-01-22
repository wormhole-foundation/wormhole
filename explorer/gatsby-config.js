require("dotenv").config({
  path: `.env.${process.env.NODE_ENV}`,
});
const siteMetadata = {
  siteUrl: process.env.GATSBY_SITE_URL,
  title: "Wormhole",
};
module.exports = {
  siteMetadata,
  plugins: [
    `gatsby-plugin-react-helmet`,
    `gatsby-plugin-top-layout`,
    `gatsby-plugin-material-ui`,
    {
      resolve: "gatsby-plugin-robots-txt",
      options: {
        host: siteMetadata.siteUrl,
        sitemap: `${siteMetadata.siteUrl}/sitemap/sitemap-index.xml`,
        env: {
          development: {
            policy: [{ userAgent: "*", disallow: ["/"] }],
          },
          production: {
            policy: [{ userAgent: "*", allow: "/" }],
          },
        },
      },
    },
    `gatsby-plugin-sitemap`,
    {
      resolve: `gatsby-plugin-google-gtag`,
      options: {
        trackingIds: [String(process.env.GATSBY_GA_TAG)],
      },
    },
    `gatsby-plugin-meta-redirect`, // make sure to put last in the array
  ],
};
