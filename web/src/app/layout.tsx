import type { Metadata, Viewport } from "next";
import { Alegreya, Alegreya_Sans } from "next/font/google";

import { ThemeProvider } from "@/components/theme-provider";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";

import "./globals.css";

const alegreya = Alegreya({
  variable: "--font-alegreya",
  subsets: ["latin"],
  display: "swap",
});

const alegreyaSans = Alegreya_Sans({
  variable: "--font-alegreya-sans",
  subsets: ["latin"],
  weight: ["400", "500", "700"],
  display: "swap",
});

export const metadata: Metadata = {
  metadataBase: new URL(process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000"),
  title: {
    default: "DnD Forge — panduan bermain dan pembuat karakter",
    template: "%s · DnD Forge",
  },
  description:
    "Panduan bermain berbahasa Indonesia, ensiklopedia ras, kelas, dan latar belakang dari SRD 5.2, serta pembuat karakter langkah demi langkah.",
  applicationName: "DnD Forge",
  openGraph: { type: "website", siteName: "DnD Forge", locale: "id_ID" },
  twitter: { card: "summary_large_image" },
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: dark)", color: "#171a23" },
    { media: "(prefers-color-scheme: light)", color: "#f4f5f7" },
  ],
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="id"
      suppressHydrationWarning
      className={`${alegreya.variable} ${alegreyaSans.variable} h-full antialiased`}
    >
      <body className="flex min-h-full flex-col">
        <ThemeProvider>
          <TooltipProvider>
            {children}
            <Toaster />
          </TooltipProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
