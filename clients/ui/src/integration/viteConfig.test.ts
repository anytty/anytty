import { describe, expect, it } from 'vitest'
import config from '../../vite.config'

describe('vite dev config', () => {
  it('does not publish the archived localweb proxy or build entry', () => {
    expect(config.server?.proxy).toBeUndefined()
    expect(config.build?.rollupOptions?.input).toBeUndefined()
  })
})
