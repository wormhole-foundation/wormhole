declare module '*.svg' {
  import React = require('react');
  export const ReactComponent: React.SFC<React.SVGProps<SVGSVGElement>>;
  const src: string;
  export default src;
}

declare module '*.jpg' {
  const jpgContent: string;
  export { jpgContent };
}

declare module '*.png' {
  const pngContent: string;
  export { pngContent };
}

declare module '*.json' {
  const jsonContent: string;
  export { jsonContent };
}
