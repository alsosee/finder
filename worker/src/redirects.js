import { REDIRECTS } from "./redirects.generated.js";

export function redirectFor(requestUrl) {
  const url = new URL(requestUrl);
  for (const rule of REDIRECTS) {
    const target = applyRedirectRule(rule, url.pathname);
    if (!target) {
      continue;
    }

    url.pathname = target;
    return Response.redirect(url.toString(), rule.status);
  }

  return null;
}

function applyRedirectRule(rule, pathname) {
  if (rule.from.endsWith("/*")) {
    const prefix = rule.from.slice(0, -1);
    if (!pathname.startsWith(prefix)) {
      return "";
    }

    const splat = pathname.slice(prefix.length);
    return rule.to.replace(":splat", splat);
  }

  if (pathname === rule.from) {
    return rule.to;
  }

  return "";
}
