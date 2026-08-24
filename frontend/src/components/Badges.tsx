import type { Difficulty, Domain } from '../api/types'
import { DOMAIN_LABELS } from '../api/types'

export function DomainBadge({ domain }: { domain: Domain }) {
  return <span className={`badge domain domain-${domain}`}>{DOMAIN_LABELS[domain]}</span>
}

export function DifficultyBadge({ difficulty }: { difficulty: Difficulty }) {
  return <span className={`badge difficulty difficulty-${difficulty}`}>{difficulty}</span>
}

export function WeightBadge({ weight }: { weight: number }) {
  return <span className="badge weight">{weight} pts</span>
}
