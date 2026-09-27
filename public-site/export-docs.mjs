import { mkdir, readdir, readFile, writeFile } from "node:fs/promises";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const source = fileURLToPath(new URL("./docs/", import.meta.url));
const output = fileURLToPath(new URL("./dist/website/docs/", import.meta.url));
const files = [];
async function walk(directory) {
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    if (entry.name.startsWith(".")) continue;
    const path = join(directory, entry.name);
    if (entry.isDirectory()) await walk(path);
    else if (entry.name.endsWith(".md")) files.push(path);
  }
}
await walk(source);
const links = [];
for (const path of files.sort()) {
  const name = relative(source, path);
  const target = join(output, name);
  await mkdir(dirname(target), { recursive: true });
  const markdown = await readFile(path, "utf8");
  await writeFile(target, markdown);
  const heading = markdown.match(/^# (.+)$/m)?.[1] ?? name;
  links.push(
    `- [${heading}](https://usage.softinator.ai/docs/${name}): ${heading}`,
  );
}
const index = `# Shared Account AI Usage Monitor documentation\n\nOpen-source, self-hosted team evidence for native AI coding subscriptions. Account-wide quota is not a per-person bill.\n\n## Pages\n\n${links.join("\n")}\n\nSource: https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor\n`;
await writeFile(join(output, "llms.txt"), index);
await writeFile(new URL("./dist/website/llms.txt", import.meta.url), index);
