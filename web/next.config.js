module.exports = {
  output: "standalone",
  reactStrictMode: true,
  images: {
    unoptimized: true,
    remotePatterns: [
      {
        protocol: "https",
        hostname: "d3i73ktnzbi69i.cloudfront.net",
      },
    ],
  },
  experimental: {
    scrollRestoration: true,
    instrumentationHook: true,
  },
  env: {
    AUTH_USERNAME: process.env.AUTH_USERNAME,
    AUTH_PASSWORD: process.env.AUTH_PASSWORD,
  },
  serverRuntimeConfig: {
    AUTH_USERNAME: process.env.AUTH_USERNAME,
    AUTH_PASSWORD: process.env.AUTH_PASSWORD,
  },
};
