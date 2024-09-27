"use client"
import React from "react";
import { usePathname } from "next/navigation";
import Link from "next/link";

const items = [
  {
    title: "Get started",
    links: [{ href: "/docs", children: "Docs" }],
  },
];

export function SideNav() {
  const pathname = usePathname();

  return (
    <nav className="sticky top-[var(--top-nav-height)] h-[calc(100vh-var(--top-nav-height))] flex-none overflow-y-auto py-10 px-8 border-r border-[var(--border-color)]">
      {items.map((item) => (
        <div key={item.title}>
          <span className="text-lg font-medium py-2 block">{item.title}</span>
          <ul className="flex flex-col p-0">
            {item.links.map((link) => {
              const active = pathname === link.href;
              return (
                <li key={link.href} className={`list-none m-0 ${active ? "active" : ""}`}>
                  <Link {...link} className="no-underline hover:underline active:underline" />
                </li>
              );
            })}
          </ul>
        </div>
      ))}
    </nav>
  );
}
