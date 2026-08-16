import mdx from '@astrojs/mdx'
import react from '@astrojs/react'
import sitemap from '@astrojs/sitemap'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'astro/config'

const customDomain = process.env.ANYTTY_SITE_CUSTOM_DOMAIN === 'true'
const base = customDomain ? '/' : (process.env.ANYTTY_SITE_BASE_PATH || '/anytty')
const site = customDomain
  ? (process.env.PUBLIC_ANYTTY_SITE_URL || 'https://anytty.com')
  : 'https://lozzo.github.io'

export default defineConfig({
  site,
  base,
  output: 'static',
  trailingSlash: 'always',
  integrations: [react(), mdx(), sitemap()],
  vite: { plugins: [tailwindcss()] },
})
