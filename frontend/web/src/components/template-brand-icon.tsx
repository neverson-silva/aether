import { Cube } from '@phosphor-icons/react'
import {
  type SimpleIcon,
  siDocker,
  siGo,
  siMariadb,
  siMongodb,
  siMysql,
  siNextdotjs,
  siNginx,
  siNodedotjs,
  siPostgresql,
  siPython,
  siReact,
  siRedis,
  siTypescript,
  siWordpress,
} from 'simple-icons'
import type { TemplateItem } from '../hooks/types'

const technologyIcons: Record<string, SimpleIcon> = {
  docker: siDocker,
  go: siGo,
  mariadb: siMariadb,
  mongodb: siMongodb,
  mysql: siMysql,
  nginx: siNginx,
  next: siNextdotjs,
  nextjs: siNextdotjs,
  node: siNodedotjs,
  nodejs: siNodedotjs,
  postgres: siPostgresql,
  postgresql: siPostgresql,
  python: siPython,
  react: siReact,
  redis: siRedis,
  typescript: siTypescript,
  wordpress: siWordpress,
}

export function resolveTemplateIcon(
  template: Pick<TemplateItem, 'icon' | 'name' | 'category'>,
) {
  const key = `${template.icon} ${template.name} ${template.category}`
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, ' ')
    .trim()
    .split(' ')
    .find((item) => technologyIcons[item])
  return key ? technologyIcons[key] : undefined
}

export function DatabaseTechnologyIcon({ engine }: { engine?: string }) {
  const icon = engine ? technologyIcons[engine.toLowerCase()] : undefined
  return icon ? (
    <svg
      aria-hidden="true"
      className="size-8"
      fill="currentColor"
      viewBox="0 0 24 24"
    >
      <path d={icon.path} />
    </svg>
  ) : (
    <Cube
      aria-hidden="true"
      size={32}
      weight="duotone"
    />
  )
}

export function TemplateBrandIcon({
  template,
  className = 'size-6',
}: {
  template: Pick<TemplateItem, 'icon' | 'name' | 'category' | 'logo_url'>
  className?: string
}) {
  if (template.logo_url)
    return (
      <img
        alt=""
        className={`${className} object-contain`}
        loading="lazy"
        src={template.logo_url}
      />
    )
  const icon = resolveTemplateIcon(template)
  return icon ? (
    <svg
      aria-hidden="true"
      className={className}
      fill="currentColor"
      viewBox="0 0 24 24"
    >
      <path d={icon.path} />
    </svg>
  ) : (
    <Cube
      aria-hidden="true"
      size={20}
    />
  )
}
