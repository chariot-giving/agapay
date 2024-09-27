import React from "react";
import Link from "next/link";

export function TableOfContents({
  toc,
}: {
  toc: Array<{ id: string; level: number; text: string }>;
}) {
  const items = toc.filter(
    (item) => item.id && (item.level === 2 || item.level === 3)
  );

  if (items.length <= 1) {
    return null;
  }

  return (
    <nav className="sticky top-[calc(2.5rem+var(--top-nav-height))] max-h-[calc(100vh-var(--top-nav-height))] flex-none self-start mb-4 pt-2 border-l border-[var(--border-color)]">
      <ul className="m-0 pl-6">
        {items.map((item) => {
          const href = `#${item.id}`;
          const active =
            typeof window !== "undefined" && window.location.hash === href;
          return (
            <li
              key={item.text}
              className={`list-none mb-4 ${
                active ? "active" : ""
              } ${item.level === 3 ? "pl-4" : ""}`}
            >
              <Link href={href} className="no-underline hover:underline active:underline">
                {item.text}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
