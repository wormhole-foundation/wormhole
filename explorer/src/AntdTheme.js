export default {
  // antd variables. see https://github.com/ant-design/ant-design/blob/master/components/style/themes/default.less
  'font-family':
    "Sora, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, 'Noto Sans', sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji'",
  'body-background': '#010114',
  'component-background': '@body-background',
  'primary-color': '#00EFD8',
  'highlight-color': '#0074FF',
  // 'processing-color': '#',
  'menu-item-font-size': '18px',
  'border-divider-color': 'darken(#808088, 20%)',
  'link-color': 'lighten(@primary-color, 20%);', // lighten for proper contrast
  'menu-dark-color': '@text-color-dark',
  // make the header the same color as the body
  'layout-header-background': '@layout-body-background',
  'menu-dark-bg': '@layout-body-background',
  'menu-dark-inline-submenu-bg': '@layout-body-background',
  'menu-inline-submenu-bg': '@layout-body-background',
  'menu-popup-bg': '@layout-body-background',

  // table styles
  'table-header-bg': '#212130',
  'table-row-hover-bg': '#212130',

  // global wormhole variables (not antd overrides)
  'max-content-width': '1400px',
  'blue-background': '#141449',
};
