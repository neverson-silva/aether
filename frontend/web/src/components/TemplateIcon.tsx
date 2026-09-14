import { useState } from "react";
import type { TemplateItem } from "../hooks";
import { TechIcon } from "./TechIcon";

export function TemplateIcon({ template, size = 32, className = "" }: { template: TemplateItem; size?: number; className?: string }) {
  const [logoFailed, setLogoFailed] = useState(false);

  if (template.logo_url && !logoFailed) {
    return (
      <img
        src={template.logo_url}
        alt=""
        width={size}
        height={size}
        loading="lazy"
        decoding="async"
        className={`object-contain ${className}`}
        onError={() => setLogoFailed(true)}
      />
    );
  }

  return <TechIcon name={template.icon || template.category} size={size} className={className} />;
}
