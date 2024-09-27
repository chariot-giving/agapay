import Link from "next/link";

export default function Navigation() {
  return (
    <nav className="flex items-center space-x-4">
      <Link href="/" className="text-gray-600 hover:text-gray-900 transition-colors duration-200">Docs</Link>
      <Link href="/api" className="text-gray-600 hover:text-gray-900 transition-colors duration-200">API Reference</Link>
    </nav>
  );
}
