const path = require("path");

module.exports = {
  output: "standalone",
  reactStrictMode: true,
  // caps stale-while-revalidate at expireTime minus revalidate, so the edge
  // never serves a page much older than its revalidate window
  expireTime: 7200,
  images: {
    unoptimized: true,
    remotePatterns: [
      {
        protocol: "https",
        hostname: "i.analogdb.com",
      },
      {
        protocol: "https",
        hostname: "d3i73ktnzbi69i.cloudfront.net",
      },
    ],
  },
  experimental: {
    scrollRestoration: true,
    optimizePackageImports: ["@mantine/core"],
  },
  webpack: (config) => {
    config.resolve.alias = {
      ...config.resolve.alias,
      "@lib": path.resolve(__dirname, "lib"),
      "@components": path.resolve(__dirname, "components"),
      "@providers": path.resolve(__dirname, "providers"),
      "@styles": path.resolve(__dirname, "styles"),
      "@hooks": path.resolve(__dirname, "hooks"),
      "@app": path.resolve(__dirname, "app"),
    };
    return config;
  },
};
