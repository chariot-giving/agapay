import { readFile } from "fs/promises";
import path from "path";
import { parseMarkdown } from "@/lib/markdoc";

export default async function Home() {
  const filePath = path.join(process.cwd(), "README.md");
  const fileContent = await readFile(filePath, "utf8");
  const content = await parseMarkdown(fileContent);

  return (
    <>
      <div className="content p-4" dangerouslySetInnerHTML={{ __html: content }} />
    </>
  );
}
