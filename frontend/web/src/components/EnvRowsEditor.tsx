import { VariableEditor, type VariableRow } from "@aether/design-system";

export interface EnvRowInput {
  key: string;
  value: string;
  secret: boolean;
}

function parseEnv(): VariableRow[] | undefined {
  const pasted = window.prompt("Paste .env content (KEY=value per line):");
  if (!pasted) return;
  return pasted.split("\n").flatMap((raw, index) => {
    const line = raw.trim();
    if (!line || line.startsWith("#")) return [];
    const match = line.match(/^([A-Za-z_][A-Za-z0-9_]*)=(.*)$/);
    if (!match) return [];
    return [{
      id: `imported-${index}-${match[1]}`,
      key: match[1],
      value: match[2].replace(/^"|"$/g, ""),
      secret: /password|secret|key|token/i.test(match[1]),
    }];
  });
}

export function EnvRowsEditor({
  value,
  onChange,
  compact: _compact,
  groups: _groups,
}: {
  value: EnvRowInput[];
  onChange: (rows: EnvRowInput[]) => void;
  compact?: boolean;
  groups?: unknown[];
}) {
  const variables: VariableRow[] = value.map((row, index) => ({
    id: `wizard-${index}-${row.key}`,
    key: row.key,
    value: row.value,
    secret: row.secret,
  }));

  return (
    <VariableEditor
      variables={variables}
      onChange={(next) => onChange(next.map(({ key, value: variableValue, secret }) => ({ key, value: variableValue, secret: Boolean(secret) })))}
      onImport={parseEnv}
    />
  );
}
