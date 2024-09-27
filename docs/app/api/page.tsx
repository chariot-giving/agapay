import yaml from "yaml";
import { readFile } from "fs/promises";
import path from "path";

import SwaggerUI from 'swagger-ui-react';
import 'swagger-ui-react/swagger-ui.css';

export default async function ApiReferencePage() {
  const filePath = path.join(process.cwd(), "public", "openapi.yaml");
  const fileContent = await readFile(filePath, "utf8");
  const spec = yaml.parse(fileContent);

  return <SwaggerUI spec={spec} />;
}
