import type { Metadata } from "next";
import { Analytics } from "@vercel/analytics/react";
import SiteHeader from "@/components/SiteHeader";
import SiteFooter from "@/components/SiteFooter";
import { brand } from "@/lib/brand";
import "./tokens.css";
import "./globals.css";

export const metadata: Metadata = {
  title: { default: `${brand.name} | Skills, agents & tools`, template: `%s | ${brand.name}` },
  description: brand.description
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" data-theme="dark" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: `try{var t=localStorage.getItem(${JSON.stringify(brand.themeKey)});if(t==='light'||t==='dark')document.documentElement.dataset.theme=t}catch(e){}` }} />
      </head>
      <body>
        <a href="#main" className="skip-link">Skip to content</a>
        <SiteHeader />
        {children}
        <SiteFooter />
        <Analytics />
      </body>
    </html>
  );
}
