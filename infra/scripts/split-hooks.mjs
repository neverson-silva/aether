// split-hooks.js — divide web/src/api/hooks.ts em arquivos individuais em
// web/src/hooks/ (kebab-case, um hook por arquivo) + tipos/qk/eventos
// compartilhados, e gera index.ts de re-export. Usa o parser do TypeScript.
import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const ts = require(path.resolve("web/node_modules/typescript/lib/typescript.js"));

const SRC = "web/src/api/hooks.ts";
const OUT = "web/src/hooks";

const src = fs.readFileSync(SRC, "utf8");
const sf = ts.createSourceFile(SRC, src, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);

const hooks = [];
const types = [];
const others = [];
let getServerSetServer = false;

for (const stmt of sf.statements) {
  const text = stmt.getFullText(sf).trim();
  if (!text) continue;
  const name =
    (stmt.name && stmt.name.getText(sf)) ||
    (ts.isVariableStatement(stmt) && stmt.declarationList.declarations[0].name.getText(sf)) ||
    "";
  if ((ts.isFunctionDeclaration(stmt) || ts.isVariableStatement(stmt)) && name.startsWith("use")) {
    hooks.push({ name, body: text });
  } else if (ts.isInterfaceDeclaration(stmt) || ts.isTypeAliasDeclaration(stmt)) {
    types.push(text);
  } else if (ts.isVariableStatement(stmt) && name === "qk") {
    others.push({ name, file: "query-keys.ts", body: text });
  } else if (ts.isVariableStatement(stmt) && name === "WEBHOOK_EVENTS") {
    others.push({ name, file: "webhook-events.ts", body: text });
  } else if (ts.isImportDeclaration(stmt) || ts.isImportEqualsDeclaration(stmt)) {
    continue; // imports do arquivo original não migram
  } else if (ts.isExportDeclaration(stmt) && text.includes("getServer")) {
    getServerSetServer = true;
  } else {
    throw new Error("declaração não classificada: " + text.slice(0, 80));
  }
}

const localTypeNames = new Set(types.map((t) => (t.match(/(?:interface|type) ([A-Za-z0-9_]+)/) || [])[1]));

const TYPES_FROM_API = [
  "ApiKey", "App", "AppDetail", "Backup", "CronJob", "Database", "Deployment", "Domain",
  "ResolvedVariable", "LoginResponse", "Me", "Member", "AuditLog", "OrgDetail", "OrgMember",
  "NotificationChannel", "Preview", "Project", "S3Destination", "Stats", "Template",
  "TimelineEvent", "Worker",
];
const CLIENT_IMPORTS = [
  "apiDelete", "apiGet", "apiPatch", "apiPost", "apiPut", "getServer", "setServer", "ApiError",
];
const REACT_IMPORTS = ["useEffect", "useState"];
const TANSTACK_IMPORTS = ["useMutation", "useQuery", "useQueryClient"];
const QK = "qk";

function kebab(name) {
  return name
    .replace(/^use/, "use-")
    .replace(/([a-z0-9])([A-Z])/g, "$1-$2")
    .toLowerCase();
}

function identifiers(text) {
  const ids = new Set();
  for (const id of text.match(/[A-Za-z_$][A-Za-z0-9_$]*/g) || []) ids.add(id);
  return ids;
}

function importBlock(ids) {
  const lines = [];
  const react = REACT_IMPORTS.filter((i) => ids.has(i));
  if (react.length) lines.push(`import { ${react.join(", ")} } from "react";`);
  const tsx = TANSTACK_IMPORTS.filter((i) => ids.has(i));
  if (tsx.length) lines.push(`import { ${tsx.join(", ")} } from "@tanstack/react-query";`);
  const client = CLIENT_IMPORTS.filter((i) => ids.has(i));
  if (client.length) lines.push(`import { ${client.join(", ")} } from "../api/client";`);
  const apiTypes = TYPES_FROM_API.filter((i) => ids.has(i));
  if (apiTypes.length) lines.push(`import type { ${apiTypes.join(", ")} } from "../api/types";`);
  const localTypes = [...localTypeNames].filter((i) => ids.has(i));
  if (localTypes.length) lines.push(`import type { ${localTypes.join(", ")} } from "./types";`);
  if (ids.has(QK)) lines.push(`import { qk } from "./query-keys";`);
  return lines.join("\n");
}

fs.rmSync(OUT, { recursive: true, force: true });
fs.mkdirSync(OUT, { recursive: true });

const indexExports = [];
for (const h of hooks) {
  const file = kebab(h.name) + ".ts";
  const content = `${importBlock(identifiers(h.body))}\n\n${h.body}\n`;
  fs.writeFileSync(path.join(OUT, file), content);
  indexExports.push(`export * from "./${kebab(h.name)}";`);
}

fs.writeFileSync(path.join(OUT, "types.ts"), types.join("\n\n") + "\n");
for (const o of others) {
  // qk e WEBHOOK_EVENTS não dependem de nada
  fs.writeFileSync(path.join(OUT, o.file), `${o.body}\n`);
}

const index = [
  `export * from "./types";`,
  `export { qk } from "./query-keys";`,
  `export { WEBHOOK_EVENTS } from "./webhook-events";`,
  ...indexExports,
];
if (getServerSetServer) index.push(`export { getServer, setServer } from "../api/client";`);
fs.writeFileSync(path.join(OUT, "index.ts"), index.join("\n") + "\n");

console.log(`hooks: ${hooks.length} | tipos: ${types.length} | outros: ${others.length}`);
