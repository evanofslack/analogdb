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
  },
  env: {
    AUTH_USERNAME: process.env.AUTH_USERNAME,
    AUTH_PASSWORD: process.env.AUTH_PASSWORD,
  },
  publicRuntimeConfig: {
    AUTH_USERNAME: process.env.AUTH_USERNAME,
    AUTH_PASSWORD: process.env.AUTH_PASSWORD,
  },
};
