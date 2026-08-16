import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { createElement } from 'react'
import { Spinner } from '../ui/spinner'

describe('loading spinner', () => {
  it('uses one circular Lucide spinner and respects reduced motion', () => {
    render(createElement(Spinner, { 'aria-label': 'Loading', role: 'status' }))

    const spinner = screen.getByRole('status', { name: 'Loading' })
    expect(spinner.tagName).toBe('svg')
    expect(spinner.classList.contains('animate-spin')).toBe(true)
    expect(spinner.classList.contains('motion-reduce:animate-none')).toBe(true)
  })
})
