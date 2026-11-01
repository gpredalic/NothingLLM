/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  async rewrites() {
    const apiTarget = process.env.API_PROXY_TARGET || "http://api:8080";
    return [
      {
        source: "/api/health",
        destination: `${apiTarget}/health`,
      },
      {
        source: "/api/:path*",
        destination: `${apiTarget}/api/:path*`,
      },
    ];
  },
};

module.exports = nextConfig;