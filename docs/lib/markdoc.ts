import Markdoc from "@markdoc/markdoc";

export async function parseMarkdown(content: string): Promise<string> {
  const ast = Markdoc.parse(content);
  const result = Markdoc.transform(ast);
  return Markdoc.renderers.html(result);
}