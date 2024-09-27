"use client";

import dynamic from "next/dynamic";
import "swagger-ui-react/swagger-ui.css";

const SwaggerUI = dynamic(() => import("swagger-ui-react"), { ssr: false });

export default function ApiReference({ spec }: { spec: any }) {
  return <SwaggerUI spec={spec} />;
}
