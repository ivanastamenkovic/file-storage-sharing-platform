// Mirrors the users service rules so people get feedback before submitting.
export const PASSWORD_RULES = [
  { id: 'length', label: 'At least 8 characters', test: (p) => [...p].length >= 8 },
  { id: 'upper', label: 'An uppercase letter', test: (p) => /\p{Lu}/u.test(p) },
  { id: 'lower', label: 'A lowercase letter', test: (p) => /\p{Ll}/u.test(p) },
  { id: 'digit', label: 'A digit', test: (p) => /\p{Nd}/u.test(p) },
  { id: 'special', label: 'A special character', test: (p) => /[\p{P}\p{S}\s]/u.test(p) },
]

export function isStrongPassword(p) {
  return PASSWORD_RULES.every((r) => r.test(p)) && new TextEncoder().encode(p).length <= 72
}
