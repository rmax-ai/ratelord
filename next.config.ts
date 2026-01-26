import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: 'export',
  basePath: '/ratelord',
  distDir: 'docs',
};

export default nextConfig;
