import * as React from "react";

export function Callout({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col p-3 bg-[#f6f9fc] border border-[#dce6e9] rounded">
      <strong>{title}</strong>
      <span>{children}</span>
    </div>
  );
}
