import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  images: { unoptimized: true },
  // Dev-only: proxy API/object requests to the local gateway so the browser
  // stays same-origin (no CORS). Not applied to the production server build.
  async rewrites() {
    if (process.env.NODE_ENV !== "development") return [];
    const target = process.env.RESTICK_GATEWAY ?? "http://localhost:1130";
    return [
      { source: "/api/:path*", destination: `${target}/api/:path*` },
      { source: "/v1/:path*", destination: `${target}/v1/:path*` },
      { source: "/docs/:path*", destination: `${target}/docs/:path*` },
    ];
  },
  async headers() {
    return [
      {
        source: "/healthz",
        headers: [{ key: "Content-Type", value: "application/json" }],
      },
    ];
  },
};

export default nextConfig;