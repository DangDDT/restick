import type { NextConfig } from "next";

// The Go service owns the HTTP origin: it serves this exported directory, the
// API under /api and /v1, and /docs. A Next runtime never serves production.
const nextConfig: NextConfig = {
  output: "export",
  images: { unoptimized: true },
};

export default nextConfig;
