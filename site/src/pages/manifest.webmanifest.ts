import type { APIRoute } from 'astro'
import { sitePath } from '@/lib/site'

export const prerender = true

export const GET: APIRoute = () => new Response(JSON.stringify({
  name: 'AnyTTY',
  short_name: 'AnyTTY',
  description: 'Open-source remote terminals and files.',
  start_url: sitePath('/'),
  display: 'standalone',
  background_color: '#ffffff',
  theme_color: '#ffffff',
  icons: [{ src: sitePath('/assets/app-icon.webp'), sizes: '1024x1024', type: 'image/webp' }],
}), { headers: { 'Content-Type': 'application/manifest+json' } })
