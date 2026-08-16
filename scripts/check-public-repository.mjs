import { access, readdir, readFile, stat } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const siteOnly = process.argv.includes("--site-only");
const failures = [];
const requiredRoot = [".gitattributes", ".gitignore", "README.md", "README.zh-CN.md", "LICENSE", "NOTICE", "THIRD_PARTY_NOTICES.txt", "CONTRIBUTING.md", "SECURITY.md", "CODE_OF_CONDUCT.md", "SUPPORT.md", "GOVERNANCE.md", "TRADEMARKS.md", "CHANGELOG.md", "RELEASE_CHECKLIST.md"];
const forbiddenPaths = ["cloud/controller", "cloud/edge", "cloud/web", "cloud/deploy", "cloud/integration", "cmd/anytty-cloud-controller", "cmd/anytty-cloud-edge", "deploy", "migrations"];
const allowedCloudPackages = new Set(["client", "daemon", "protocol", "securetransport", "ticket"]);

async function exists(file) { try { await access(file); return true; } catch { return false; } }
async function filesBelow(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const nested = await Promise.all(entries.filter((entry) => entry.name !== ".git" && entry.name !== "node_modules").map(async (entry) => {
    const target = path.join(directory, entry.name);
    return entry.isDirectory() && ![".astro", "dist"].includes(entry.name) ? filesBelow(target) : entry.isDirectory() ? [] : [target];
  }));
  return nested.flat();
}

if (!siteOnly) {
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
  for (const file of markdownFiles) {
    const markdown = await readFile(file, "utf8");
    for (const match of markdown.matchAll(/\[[^\]]+\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g)) {
      const raw = match[1].replace(/^<|>$/g, "");
      if (/^(?:https?:|mailto:|#|\/)/.test(raw) || raw.includes("{base}")) continue;
      const decoded = decodeURIComponent(raw.split("#", 1)[0]);
      const target = path.resolve(path.dirname(file), decoded);
      if (!(await exists(target))) failures.push(`${path.relative(root, file)} has broken local Markdown link: ${raw}`);
    }
  }
}

const siteRoot = path.join(root, "site", "dist");
if (!(await exists(siteRoot))) failures.push("site artifact is missing; run npm run site:build");
else {
  const siteFiles = await filesBelow(siteRoot);
  const htmlFiles = siteFiles.filter((file) => file.endsWith(".html"));
  if (htmlFiles.length < 18) failures.push(`expected English and Chinese site pages, found ${htmlFiles.length}`);
  for (const name of ["robots.txt", "sitemap-index.xml", "manifest.webmanifest", ".nojekyll"]) if (!(await exists(path.join(siteRoot, name)))) failures.push(`site artifact is missing SEO/Pages file: ${name}`);
  const cssFiles = siteFiles.filter((file) => file.endsWith(".css"));
  const css = (await Promise.all(cssFiles.map((file) => readFile(file, "utf8")))).join("\n");
  if (!css.includes("prefers-reduced-motion")) failures.push("site CSS is missing reduced-motion behavior");
  for (const file of htmlFiles) {
    const html = await readFile(file, "utf8");
    const relative = path.relative(siteRoot, file);
    const normalizedHtml = html.toLowerCase();
    for (const required of ["<!doctype html>", "<html lang=", "<title>", "<meta name=\"description\"", "<header", "<main", "<footer"]) if (!normalizedHtml.includes(required)) failures.push(`${relative} missing HTML structure: ${required}`);
    if (!html.includes("href=\"#main\"") || !html.includes("id=\"main\"")) failures.push(`${relative} missing skip-link target`);
    const ids = [...html.matchAll(/\sid="([^"]+)"/g)].map((match) => match[1]);
    if (new Set(ids).size !== ids.length) failures.push(`${relative} has duplicate ids`);
    for (const match of html.matchAll(/\s(?:href|src)="([^"]+)"/g)) {
      const url = match[1];
      if (/^(?:https?:|#|mailto:)/.test(url)) continue;
      if (url.startsWith("data:") || url.startsWith("javascript:")) continue;
      const cleanUrl = url.split(/[?#]/, 1)[0];
      const configuredBase = `${process.env.ANYTTY_SITE_BASE_PATH ?? "/anytty"}/`.replace("//", "/");
      const local = cleanUrl.startsWith(configuredBase) ? cleanUrl.slice(configuredBase.length) : cleanUrl.startsWith("/") ? cleanUrl.slice(1) : path.relative(siteRoot, path.resolve(path.dirname(file), cleanUrl));
      const target = path.join(siteRoot, local || "index.html");
      if (!(await exists(target)) || (await exists(target) && (await stat(target)).isDirectory() && !(await exists(path.join(target, "index.html"))))) failures.push(`${relative} has broken local link: ${url}`);
    }
  }
  const englishHome = await readFile(path.join(siteRoot, "index.html"), "utf8");
  const chineseHome = await readFile(path.join(siteRoot, "zh-CN", "index.html"), "utf8");
  if (!englishHome.includes('<html lang="en"')) failures.push("English home page has the wrong document language");
  if (!chineseHome.includes('<html lang="zh-CN"')) failures.push("Chinese home page has the wrong document language");
  for (const required of ["rel=\"canonical\"", "hreflang=\"zh-CN\"", "application/ld+json"]) if (!englishHome.toLowerCase().includes(required.toLowerCase())) failures.push(`English home page is missing SEO markup: ${required}`);
}

if (failures.length) {
  console.error(failures.map((failure) => `- ${failure}`).join("\n"));
  process.exit(1);
}
console.log(siteOnly ? "site structure and links passed" : "public repository boundary, structure, and links passed");
