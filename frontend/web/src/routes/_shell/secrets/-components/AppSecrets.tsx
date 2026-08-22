import { Badge } from "@aether/design-system";
import { useAppDetail } from "../../../../hooks";

export function AppSecrets({ appId, name }: { appId: string; name: string }) {
  const { data: detail } = useAppDetail(appId);
  const secrets = (detail?.env ?? []).filter((e) => e.secret);
  if (!secrets.length) return null;
  return (
    <div className="border border-outline-variant/60 rounded p-sm">
      <div className="flex items-center justify-between mb-sm">
        <span className="font-body-md text-body-md text-on-surface">{name}</span>
        <Badge tone="neutral">{secrets.length} secrets</Badge>
      </div>
        <table className="w-full text-left"><thead><tr className="text-label-caps text-muted-foreground"><th className="px-2 py-2 font-semibold">Name</th><th className="px-2 py-2 font-semibold">Value</th></tr></thead><tbody>
        {secrets.map((s) => (
          <tr key={s.name}>
            <td className="px-sm py-1.5 font-code-md text-code-md text-on-surface">{s.name}</td>
            <td className="px-sm py-1.5 font-code-md text-code-md text-on-surface-variant/60">
              {String.fromCharCode(0x2022).repeat(10)}
            </td>
          </tr>
        ))}
        </tbody></table>
    </div>
  );
}
