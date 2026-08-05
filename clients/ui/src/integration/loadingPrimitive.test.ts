import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const appStylesSource = readFileSync(resolve(process.cwd(), 'src/entries/appStyles.css'), 'utf8')

function cssBlock(selector: string, containing?: string): string {
  const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const matches = [...appStylesSource.matchAll(new RegExp(`${escaped}\\s*\\{([^}]+)\\}`, 'gs'))]
  const match = containing ? matches.find((candidate) => candidate[1]?.includes(containing)) ?? null : matches[0] ?? null
  expect(match, `missing CSS block for ${selector}`).not.toBeNull()
  return match?.[1] ?? ''
}

describe('square loading primitive', () => {
  it('keeps the frame fixed while one masked segment travels along its perimeter', () => {
    const frame = cssBlock('.anytty-square-spinner')
    const fixedPerimeter = cssBlock('.anytty-square-spinner::before', 'color-mix(')
    const segment = cssBlock('.anytty-square-spinner::after', 'conic-gradient(')
    const keyframes = cssBlock('to', '--anytty-square-progress-angle: 360deg')

    expect(frame).toContain('height: 16px')
    expect(frame).toContain('width: 16px')
    expect(frame).toContain('box-sizing: border-box')
    expect(frame).not.toMatch(/animation|transform|rotate/)
    expect(fixedPerimeter).toContain('color-mix(')
    expect(fixedPerimeter).not.toMatch(/animation|transform|rotate/)

    expect(appStylesSource).toContain('@property --anytty-square-progress-angle')
    expect(segment).toContain('conic-gradient(')
    expect(appStylesSource).toMatch(
      /\.anytty-square-spinner::before,\s*\.anytty-square-spinner::after\s*\{[^}]+inset: 0[^}]+mask-composite: exclude[^}]+-webkit-mask-composite: xor/s,
    )
    expect(segment).not.toMatch(/transform|rotate/)
    expect(keyframes).toContain('--anytty-square-progress-angle: 360deg')
    expect(keyframes).not.toMatch(/transform|rotate/)
  })

  it('stops the perimeter segment when reduced motion is requested', () => {
    expect(appStylesSource).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]+?\.anytty-square-spinner::after[\s\S]+?animation: none !important/,
    )
  })
})
