const path = require('path');

module.exports = {
  output: 'standalone',
  reactStrictMode: true,
  images: {
    unoptimized: true,
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'd3i73ktnzbi69i.cloudfront.net',
      },
    ],
  },
  experimental: {
    scrollRestoration: true,
    optimizePackageImports: ['@mantine/core'],
  },
  env: {
    AUTH_USERNAME: process.env.AUTH_USERNAME,
    AUTH_PASSWORD: process.env.AUTH_PASSWORD,
  },
  serverRuntimeConfig: {
    AUTH_USERNAME: process.env.AUTH_USERNAME,
    AUTH_PASSWORD: process.env.AUTH_PASSWORD,
  },
  webpack: (config) => {
    config.resolve.alias = {
      ...config.resolve.alias,
      '@lib': path.resolve(__dirname, 'lib'),
      '@components': path.resolve(__dirname, 'components'),
      '@providers': path.resolve(__dirname, 'providers'),
      '@styles': path.resolve(__dirname, 'styles'),
      '@hooks': path.resolve(__dirname, 'hooks'),
    };
    return config;
  },
};
