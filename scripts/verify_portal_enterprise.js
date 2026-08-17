const fs = require("fs");

const htmlPath = "web/index.html";
const html = fs.readFileSync(htmlPath, "utf8");

const scripts = [...html.matchAll(/<script>([\s\S]*?)<\/script>/g)].map((m) => m[1]);
for (const script of scripts) {
  new Function(script);
}

const visibleText = html
  .replace(/<script>[\s\S]*?<\/script>/g, " ")
  .replace(/<style>[\s\S]*?<\/style>/g, " ")
  .replace(/<[^>]+>/g, " ")
  .replace(/\s+/g, " ")
  .trim();

const bannedVisible = [
  "local demo",
  "Demo Assistant",
  "Prepare Demo",
  "What To Say",
  "what to say",
  "this you can say",
  "azureblob-local",
  "everest_demo",
  "everest-demo",
  "dry-run",
  "dry_run",
  "Item 13",
];

const bannedSource = [
  "azureblob-local",
  "local demo",
  "Demo Assistant",
  "Prepare Demo",
  "What To Say",
  "this you can say",
  "\u00e2",
  "\u02dc",
  "\u00be",
  "\u2020",
];

const requiredSource = [
  "Operations Assistant",
  "Add Azure Blob",
  "notificationDrawer",
  "clearNotifications",
  "wbSelectedContainers",
  "friendlyActionMode",
  "sanitizePortalText",
  "Account access key",
];

const failures = [];
for (const term of bannedVisible) {
  if (visibleText.toLowerCase().includes(term.toLowerCase())) {
    failures.push(`Visible banned term: ${term}`);
  }
}
for (const term of bannedSource) {
  if (html.includes(term)) {
    failures.push(`Source banned term: ${term}`);
  }
}
for (const term of requiredSource) {
  if (!html.includes(term)) {
    failures.push(`Required portal feature missing: ${term}`);
  }
}

if (failures.length) {
  console.error(JSON.stringify({ ok: false, failures }, null, 2));
  process.exit(1);
}

console.log(JSON.stringify({ ok: true, scripts: scripts.length, checked: htmlPath }));
