"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { brand, moduleNavigation } from "@/lib/brand";
import FamilyNavigation from "@/components/FamilyNavigation";

export default function SiteHeader() {
  const pathname = usePathname();
  const menu = useRef<HTMLDetailsElement>(null);
  const sessionTheme = useRef<string | null>(null);
  const [theme, setTheme] = useState("dark");

  useLayoutEffect(() => {
    // Error-route hydration can restore the server's default html attribute.
    // Initialize from storage once; later routes must preserve choices even
    // when storage writes fail and the persisted preference is stale.
    let next = sessionTheme.current;
    if (next === null) {
      next = document.documentElement.dataset.theme === "light" ? "light" : "dark";
      try {
        const saved = localStorage.getItem(brand.themeKey);
        if (saved === "light" || saved === "dark") next = saved;
      } catch { /* Storage is optional. */ }
    }
    sessionTheme.current = next;
    document.documentElement.dataset.theme = next;
    setTheme(next);
  }, [pathname]);

  useEffect(() => {
    function syncTheme(event: StorageEvent) {
      if (event.key !== brand.themeKey && event.key !== null) return;
      const next = event.newValue === "light" ? "light" : "dark";
      sessionTheme.current = next;
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
    sessionTheme.current = next;
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
        <Link href="/" className="wordmark" aria-label={`${brand.name} — ${brand.siteName} home`}>
          <span>{brand.name}</span><span className="wordmark-site">{brand.siteName}</span>
        </Link>
        <nav className="desktop-nav" aria-label="Catalogue">{links}</nav>
        <button className="theme-toggle" type="button" aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} mode`} onClick={toggleTheme}>
          {theme === "dark" ? "Light mode" : "Dark mode"}
        </button>
        <details className="mobile-menu" ref={menu} onKeyDown={(event) => {
          if (event.key === "Escape") { closeMenu(true); event.stopPropagation(); }
        }} onBlur={(event) => {
          if (!event.currentTarget.contains(event.relatedTarget as Node | null)) closeMenu();
        }}>
          <summary>Menu</summary>
          <nav aria-label="Catalogue">{links}</nav>
        </details>
      </div>
      <div className="header-family">
        <p>{brand.descriptor}</p>
        <FamilyNavigation label="AI Knowledge Hub sites" />
      </div>
    </header>
  );
}
