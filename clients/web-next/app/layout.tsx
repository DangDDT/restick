import type { Metadata, Viewport } from "next";
import { Be_Vietnam_Pro, Bricolage_Grotesque, JetBrains_Mono } from "next/font/google";
import "./globals.css";
import { Providers } from "./providers";

const ui = Be_Vietnam_Pro({
  subsets: ["latin", "latin-ext", "vietnamese"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-ui",
  display: "swap",
});
const head = Bricolage_Grotesque({
  subsets: ["latin", "latin-ext", "vietnamese"],
  weight: ["500", "600", "700"],
  variable: "--font-head",
  display: "swap",
});
const code = JetBrains_Mono({
  subsets: ["latin", "latin-ext", "vietnamese"],
  variable: "--font-code",
  display: "swap",
});

export const metadata: Metadata = {
  title: { default: "Restick Studio", template: "%s · Restick Studio" },
  description: "Làm sticker dễ như chơi: soạn prompt, tạo pack và cấp API key cho app.",
  applicationName: "Restick Studio",
  manifest: "/manifest.webmanifest",
  icons: {
    icon: [
      { url: "/icon.svg", type: "image/svg+xml" },
      { url: "/favicon.ico", sizes: "any" },
    ],
    apple: "/apple-touch-icon.png",
  },
  // metadataBase is unknown at build time (the public origin is runtime config),
  // so the social image is a root-relative path that crawlers resolve per host.
  openGraph: { title: "Restick Studio", description: "Làm sticker dễ như chơi.", siteName: "Restick", images: ["/og-image.png"] },
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#FDF3F9" },
    { media: "(prefers-color-scheme: dark)", color: "#2a2433" },
  ],
};

// Applies a stored theme before first paint so there is no light/dark flash.
const THEME_BOOT = `try{var t=localStorage.getItem("restick_theme");if(t==="light"||t==="dark")document.documentElement.dataset.theme=t}catch(e){}`;

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="vi" className={`${ui.variable} ${head.variable} ${code.variable} h-full`} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: THEME_BOOT }} />
      </head>
      <body className="min-h-full">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
