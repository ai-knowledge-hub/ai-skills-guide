"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { brand, moduleNavigation } from "@/lib/brand";

export default function SiteHeader() {
  const pathname = usePathname();
  const menu = useRef<HTMLDetailsElement>(null);
  const [theme, setTheme] = useState("dark");

  useEffect(() => {
    setTheme(document.documentElement.dataset.theme === "light" ? "light" : "dark");
    function syncTheme(event: StorageEvent) {
      if (event.key !== brand.themeKey && event.key !== null) return;
      const next = event.newValue === "light" ? "light" : "dark";
      document.documentElement.dataset.theme = next;
      setTheme(next);
    }
    window.addEventListener("storage", syncTheme);
    return () => window.removeEventListener("storage", syncTheme);
  }, []);

  useEffect(() => {
    if (menu.current) menu.current.open = false;
  }, [pathname]);

  function closeMenu(returnFocus = false) {
    if (!menu.current) return;
    menu.current.open = false;
    if (returnFocus) menu.current.querySelector("summary")?.focus();
  }

  function toggleTheme() {
    const next = theme === "dark" ? "light" : "dark";
    document.documentElement.dataset.theme = next;
    setTheme(next);
    try { localStorage.setItem(brand.themeKey, next); } catch { /* Storage is optional. */ }
  }

  const links = moduleNavigation.map(({ href, label }) => (
    <Link key={href} href={href} aria-current={pathname.startsWith(href) ? "page" : undefined} onClick={() => closeMenu()}>
      {label}
    </Link>
  ));

  return (
    <header className="site-header">
      <div className="header-inner">
        <Link href="/" className="wordmark" aria-label={`${brand.name} home`}>{brand.name}</Link>
        <span className="header-descriptor">The open-source catalogue</span>
        <nav className="desktop-nav" aria-label="Main navigation">{links}</nav>
        <button className="theme-toggle" type="button" aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} mode`} onClick={toggleTheme}>
          {theme === "dark" ? "Light mode" : "Dark mode"}
        </button>
        <details className="mobile-menu" ref={menu} onKeyDown={(event) => {
          if (event.key === "Escape") { closeMenu(true); event.stopPropagation(); }
        }} onBlur={(event) => {
          if (!event.currentTarget.contains(event.relatedTarget as Node | null)) closeMenu();
        }}>
          <summary>Menu</summary>
          <nav aria-label="Mobile navigation">{links}<a href={brand.publication}>The publication</a></nav>
        </details>
      </div>
    </header>
  );
}
