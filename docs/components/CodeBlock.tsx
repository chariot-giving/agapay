import * as React from 'react';

interface CodeBlockProps {
  children: React.ReactNode;
  'data-language': string;
}

export function CodeBlock({ children, 'data-language': language }: CodeBlockProps) {
  return (
    <div className="relative" aria-live="polite">
      <pre
        className={`bg-gray-100 p-4 rounded-lg overflow-x-auto ${getLanguageClass(language)}`}
      >
        <code className="text-sm font-mono">
          {children}
        </code>
      </pre>
    </div>
  );
}

function getLanguageClass(language: string): string {
  switch (language) {
    case 'javascript':
    case 'js':
      return 'text-yellow-700';
    case 'typescript':
    case 'ts':
      return 'text-blue-700';
    case 'python':
      return 'text-green-700';
    case 'html':
      return 'text-red-700';
    case 'css':
      return 'text-pink-700';
    default:
      return 'text-gray-700';
  }
}