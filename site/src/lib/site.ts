export type Locale = "en" | "zh-CN";

const configuredBase = import.meta.env.BASE_URL.replace(/\/$/, "");
export const basePath = configuredBase === "/" ? "" : configuredBase;
export const siteUrl = (import.meta.env.PUBLIC_ANYTTY_SITE_URL ?? (basePath ? "https://anytty.github.io/anytty" : "https://anytty.com")).replace(/\/$/, "");

export function sitePath(pathname: string) {
  const path = pathname.startsWith("/") ? pathname : `/${pathname}`;
  return `${basePath}${path}` || "/";
}

export function canonicalUrl(pathname: string) {
  const path = pathname === "/" ? "" : pathname.replace(/\/$/, "");
  return `${siteUrl}${path}`;
}

export const docs = [
  ["quick-start", "Quick start", "快速开始"],
  ["connections", "Connections", "连接方式"],
  ["pairing", "QR pairing", "扫码配对"],
  ["terminals", "Terminal management", "终端管理"],
  ["files", "File management", "文件管理"],
  ["cloud", "Cloud", "Cloud"],
  ["security", "Security model", "安全模型"],
  ["build-contribute", "Build & contribute", "构建与贡献"],
] as const;

export function localizedPath(locale: Locale, pathname = "/") {
  const suffix = pathname === "/" ? "/" : `${pathname.replace(/\/$/, "")}/`;
  return locale === "zh-CN" ? `/zh-CN${suffix}` : suffix;
}
