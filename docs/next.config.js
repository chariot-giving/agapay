// next.config.js

const withMarkdoc = require('@markdoc/next.js');

module.exports = withMarkdoc(/* options */)({
  pageExtensions: ['ts', 'tsx', 'js', 'jsx', 'md'],
});