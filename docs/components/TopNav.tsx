import React from "react";
import Link from "next/link";

export function TopNav({ children }: { children: React.ReactNode }) {
  return (
    <nav className="fixed top-0 w-full z-100 flex items-center justify-between gap-4 p-4 border-b border-[var(--border-color)] bg-white">
      <Link href="/" className="flex no-underline">
        Home
      </Link>
      <section className="flex gap-4 p-0">{children}</section>
    </nav>
  );
}
