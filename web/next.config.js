module.exports = {
  output: 'standalone',
  reactStrictMode: true,
  images: {
    unoptimized: false,
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
};
