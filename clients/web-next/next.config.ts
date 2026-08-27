import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "export",
  images: { unoptimized: true },
  // Dev-only: proxy API/object requests to the local gateway so the browser
  // stays same-origin (no CORS). Ignored by `next build` (export mode).
  async rewrites() {
    if (process.env.NODE_ENV !== "development") return [];
    const target = process.env.RESTICK_GATEWAY ?? "http://localhost:1130";
    return [
      { source: "/api/:path*", destination: `${target}/api/:path*` },
      { source: "/v1/:path*", destination: `${target}/v1/:path*` },
      { source: "/docs/:path*", destination: `${target}/docs/:path*` },
    ];
  },
};

export default nextConfig;