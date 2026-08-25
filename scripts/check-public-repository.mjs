import { access, readdir, readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const failures = [];
const requiredRoot = [".gitattributes", ".gitignore", "README.md", "README.zh-CN.md", "LICENSE", "NOTICE", "THIRD_PARTY_NOTICES.txt", "CONTRIBUTING.md", "SECURITY.md", "CODE_OF_CONDUCT.md", "SUPPORT.md", "GOVERNANCE.md", "TRADEMARKS.md", "CHANGELOG.md", "RELEASE_CHECKLIST.md", "docs/SECURITY_BOUNDARY.md", "docs/zh-CN/SECURITY_BOUNDARY.md"];
const forbiddenPaths = ["cloud/controller", "cloud/edge", "cloud/web", "cloud/deploy", "cloud/integration", "cmd/anytty-cloud-controller", "cmd/anytty-cloud-edge", "deploy", "migrations"];
const allowedCloudPackages = new Set(["client", "daemon", "protocol", "securetransport", "ticket"]);
const ignoredDirectories = new Set([".git", ".artifacts", ".astro", "DerivedData", "dist", "node_modules"]);

async function exists(file) { try { await access(file); return true; } catch { return false; } }
async function filesBelow(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const nested = await Promise.all(entries.filter((entry) => !ignoredDirectories.has(entry.name)).map(async (entry) => {
    const target = path.join(directory, entry.name);
    return entry.isDirectory() ? filesBelow(target) : [target];
  }));
  return nested.flat();
}

for (const name of requiredRoot) if (!(await exists(path.join(root, name)))) failures.push(`missing release document: ${name}`);
for (const name of forbiddenPaths) if (await exists(path.join(root, name))) failures.push(`private implementation path present: ${name}`);
if (await exists(path.join(root, "cloud"))) {
  for (const entry of await readdir(path.join(root, "cloud"), { withFileTypes: true })) {
    if (!allowedCloudPackages.has(entry.name)) failures.push(`unreviewed Cloud package present: cloud/${entry.name}`);
  }
}
const publicFiles = await filesBelow(root);
const sourceFiles = publicFiles.filter((file) => /\.(?:go|ts|tsx|js|mjs|mdx)$/.test(file));
const privateImport = /github\.com\/anytty\/anytty\/(?:cloud\/(?:controller|edge|web|deploy|integration)|cmd\/anytty-cloud-)/;
const credentialPattern = /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/;
for (const file of sourceFiles) {
  const content = await readFile(file, "utf8");
  if (privateImport.test(content)) failures.push(`private implementation import: ${path.relative(root, file)}`);
  if (credentialPattern.test(content)) failures.push(`possible embedded credential: ${path.relative(root, file)}`);
}
const riskyNames = publicFiles.filter((file) => /(?:^|\/)(?:\.env|credentials?|secrets?)(?:\.[^/]*)?$/i.test(path.relative(root, file)) && !file.endsWith(".example"));
for (const file of riskyNames) failures.push(`possible sensitive configuration file: ${path.relative(root, file)}`);
const markdownFiles = publicFiles.filter((file) => /\.mdx?$/.test(file) && !file.includes(`${path.sep}.artifacts${path.sep}`));
const privateArchitecturePattern = /\b(?:PostgreSQL|EdgeControl|AgentGateway|ClientGateway|KeyBundle)\b|Cloud Controller|Cloud Edge|数据库迁移|生产部署/;
for (const file of markdownFiles) {
  const markdown = await readFile(file, "utf8");
  if (privateArchitecturePattern.test(markdown)) failures.push(`${path.relative(root, file)} discloses managed-service implementation details`);
  for (const match of markdown.matchAll(/\[[^\]]+\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g)) {
    const raw = match[1].replace(/^<|>$/g, "");
    if (/^(?:https?:|mailto:|#|\/)/.test(raw) || raw.includes("{base}")) continue;
    const decoded = decodeURIComponent(raw.split("#", 1)[0]);
    const target = path.resolve(path.dirname(file), decoded);
    if (!(await exists(target))) failures.push(`${path.relative(root, file)} has broken local Markdown link: ${raw}`);
  }
}

if (failures.length) {
  console.error(failures.map((failure) => `- ${failure}`).join("\n"));
  process.exit(1);
}
console.log("public repository boundary, structure, and links passed");
