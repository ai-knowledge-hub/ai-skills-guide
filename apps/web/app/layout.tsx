import type { Metadata } from "next";
import { Space_Grotesk, IBM_Plex_Mono } from "next/font/google";
import { Analytics } from "@vercel/analytics/react";
import "./tokens.css";
import "./globals.css";

const heading = Space_Grotesk({
  subsets: ["latin"],
  variable: "--font-heading",
  weight: ["400", "600", "700"]
});

const mono = IBM_Plex_Mono({
  subsets: ["latin"],
  variable: "--font-mono",
  weight: ["400", "500"]
});

export const metadata: Metadata = {
  title: "AI Knowledge Hub",
  description: "Open skills hub for marketing and adtech AI workflows"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${heading.variable} ${mono.variable}`}>
      <body className="theme-hybrid">
        <div className="strip">
          AI KNOWLEDGE HUB • SUPER EARLY BUILD • OPEN SOURCE SKILLS •
          CONTRIBUTE VIA PR •
        </div>
        {children}
        <Analytics />
      </body>
    </html>
  );
}
