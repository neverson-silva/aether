import type { SqlClause, SqlContext } from "./types";

// Tokenizer that respects quotes, dollar-quoted strings and parentheses so
// clause detection stays accurate while the user types.
export function tokenize(sql: string): string[] {
  const tokens: string[] = [];
  let cur = "";
  let quote: string | null = null;
  let dollar: string | null = null;
  let paren = 0;
  for (let i = 0; i < sql.length; i++) {
    const ch = sql[i];
    if (dollar) {
      if (sql.startsWith(dollar, i)) {
        i += dollar.length - 1;
        dollar = null;
      }
      continue;
    }
    if (quote) {
      if (ch === quote) {
        if (quote === "'" && sql[i + 1] === "'") {
          i++;
          continue;
        }
        quote = null;
      }
      continue;
    }
    if (ch === "$") {
      const m = /^\$[A-Za-z_0-9]*\$/.exec(sql.slice(i));
      if (m) {
        dollar = m[0];
        i += m[0].length - 1;
        continue;
      }
    }
    if (ch === "'" || ch === '"' || ch === "`") {
      quote = ch;
      if (cur) {
        tokens.push(cur);
        cur = "";
      }
      continue;
    }
    if (/[()]/.test(ch)) {
      if (ch === "(") paren++;
      else paren = Math.max(0, paren - 1);
      if (cur) {
        tokens.push(cur);
        cur = "";
      }
      continue;
    }
    if (/[\s,;=.<>+\-*/]/.test(ch)) {
      if (cur) {
        tokens.push(cur);
        cur = "";
      }
      continue;
    }
    cur += ch;
  }
  if (cur) tokens.push(cur);
  return tokens;
}

const CLAUSE_KW: Record<string, SqlClause> = {
  select: "select",
  from: "from",
  join: "join",
  inner: "join",
  left: "join",
  right: "join",
  full: "join",
  cross: "join",
  on: "on",
  where: "where",
  group: "group",
  having: "having",
  order: "order",
  limit: "limit",
  insert: "insert",
  into: "insert",
  values: "insert",
  update: "update",
  set: "set",
};

function norm(s: string): string {
  return s.toLowerCase();
}

// Extract the SQL context at the cursor: current clause, aliases and tables
// already in scope, and whether the cursor follows a dot (alias/schema.column).
export function extractContext(sql: string, cursor: number): SqlContext {
  const before = sql.slice(0, cursor);
  const lower = before.toLowerCase();
  const tokens = tokenize(before);

  let clause: SqlClause = "unknown";
  let lastSignificant = "";
  for (const t of tokens) {
    const k = CLAUSE_KW[norm(t)];
    if (k) clause = k;
    lastSignificant = t;
  }
  if (clause === "join" && norm(lastSignificant) === "on") clause = "on";
  if (clause === "unknown" && lower.includes("from") && !lower.includes("select")) clause = "from";

  // Aliases are extracted from the full query (they are often declared after
  // the cursor, e.g. `SELECT u.| FROM users u`), while clause detection uses
  // only the text before the cursor.
  const aliases: Record<string, string> = {};
  const tablesInScope: string[] = [];
  const toks = tokenize(sql);
  for (let i = 0; i < toks.length; i++) {
    const t = norm(toks[i]);
    if (t === "from" || t === "join" || t === "update" || t === "into") {
      const table = toks[i + 1];
      if (!table) continue;
      tablesInScope.push(table.toLowerCase());
      const alias = toks[i + 2];
      if (alias && !CLAUSE_KW[norm(alias)] && !/^\(/.test(alias)) {
        aliases[alias.toLowerCase()] = table.toLowerCase();
      }
    }
  }

  const dotMatch = before.match(/([a-zA-Z_][a-zA-Z0-9_$]*)\.[a-zA-Z0-9_$]*$/);
  const afterDot = !!dotMatch;
  const dotTable = dotMatch ? dotMatch[1].toLowerCase() : null;

  const wordMatch = before.match(/[a-zA-Z_][a-zA-Z0-9_$]*$/);
  const prefix = wordMatch ? wordMatch[0] : "";

  return { sql, cursor, prefix, clause, aliases, tablesInScope, afterDot, dotTable };
}

// Extract tables + join pairs referenced anywhere in a query (for learning).
export function extractPattern(sql: string): { tables: string[]; joins: [string, string][] } {
  const toks = tokenize(sql);
  const tables: string[] = [];
  const joins: [string, string][] = [];
  let prevTable: string | null = null;
  for (let i = 0; i < toks.length; i++) {
    const t = norm(toks[i]);
    if (t === "from" || t === "join" || t === "update" || t === "into") {
      const table = toks[i + 1];
      if (!table) continue;
      const name = table.toLowerCase();
      if (!tables.includes(name)) tables.push(name);
      if (prevTable && t === "join" && !(prevTable === name)) joins.push([prevTable, name]);
      prevTable = name;
    }
  }
  return { tables, joins };
}