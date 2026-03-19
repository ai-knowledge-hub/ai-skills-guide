import type { Metadata } from "next";
import { Analytics } from "@vercel/analytics/react";
import "./tokens.css";
import "./globals.css";

export const metadata: Metadata = {
  title: "AI Knowledge Hub",
  description: "Open skills hub for marketing and adtech AI workflows"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
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
