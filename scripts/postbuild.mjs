import fs from "node:fs/promises";
import path from "node:path";

const projectRoot = process.cwd();
const publicDir = path.join(projectRoot, "public");
const outDir = path.join(projectRoot, "docs");

async function copyIfExists(fileName) {
  const src = path.join(publicDir, fileName);
  const dest = path.join(outDir, fileName);

  try {
    await fs.access(src);
  } catch {
    return;
  }

  await fs.mkdir(path.dirname(dest), { recursive: true });
  await fs.copyFile(src, dest);
}

await copyIfExists("CNAME");
await copyIfExists(".nojekyll");
