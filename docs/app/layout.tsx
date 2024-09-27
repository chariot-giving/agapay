import "./globals.css";
import { Inter } from "next/font/google";
import Navigation from "@/components/Navigation";
import { SideNav, TableOfContents, TopNav } from '@/components';

// Import Prism.js and its themes
import 'prismjs';
import 'prismjs/components/prism-bash.min';
import 'prismjs/themes/prism.css';

const inter = Inter({ subsets: ["latin"] });

export const metadata = {
  title: "Agapay Documentation",
  description: "API documentation for Agapay",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="h-full">
      <body className={`${inter.className} flex flex-col min-h-screen`}>
        <TopNav>
          <Navigation />
        </TopNav>
        <div className="flex w-full flex-grow pt-[var(--top-nav-height)]">
          <main className="flex-grow text-base p-0 pr-8 pb-8">{children}</main>
          <TableOfContents toc={[]} /> {/* We'll update this in the page component */}
        </div>
        <footer className="w-full py-4 bg-gray-100 mt-auto">
          <p className="text-center text-sm text-gray-600">&copy; 2024 Chariot Giving, Inc. All rights reserved.</p>
        </footer>
      </body>
    </html>
  );
}