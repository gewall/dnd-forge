import type { NextConfig } from "next";

// Images load straight from the API's /v1/media redirect (302 → presigned
// S3 URL), so both hosts must be allowed.
const apiOrigin = new URL(process.env.API_BASE_URL ?? "https://api-dnd.gewall.my.id").origin;
const mediaOrigins = [apiOrigin, "https://s3.gewall.my.id"];

// Vercel's preview toolbar injects a script and an iframe from vercel.live.
const isPreview = process.env.VERCEL_ENV === "preview";
const vercelLive = isPreview ? " https://vercel.live" : "";

const csp = [
  "default-src 'self'",
  // Next.js inlines bootstrap scripts; a nonce would force every page dynamic.
  `script-src 'self' 'unsafe-inline'${vercelLive}`,
  "style-src 'self' 'unsafe-inline'",
  `img-src 'self' data: blob: ${mediaOrigins.join(" ")}`,
  "font-src 'self'",
  `connect-src 'self'${vercelLive}`,
  `frame-src${isPreview ? vercelLive : " 'none'"}`,
  "frame-ancestors 'none'",
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
].join("; ");

const securityHeaders = [
  { key: "Content-Security-Policy", value: csp },
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
  { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=()" },
];

const nextConfig: NextConfig = {
  cacheComponents: true,
  partialPrefetching: true,
  poweredByHeader: false,
  images: {
    // Skip Vercel image optimization: the API already serves WebP thumbnails.
    unoptimized: true,
    remotePatterns: mediaOrigins.map((origin) => new URL(`${origin}/**`)),
  },
  async headers() {
    // Dev needs eval for React Refresh; only ship the CSP in production builds.
    if (process.env.NODE_ENV !== "production") return [];
    return [{ source: "/:path*", headers: securityHeaders }];
  },
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
