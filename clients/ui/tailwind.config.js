import tailwindcssAnimate from 'tailwindcss-animate'

const themedZinc = Object.fromEntries(
  [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950]
    .map((shade) => [shade, `rgb(var(--anytty-neutral-${shade}) / <alpha-value>)`]),
)

/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: { zinc: themedZinc },
      borderRadius: {
        lg: 'var(--radius-lg)',
        md: 'var(--radius-md)',
        sm: 'var(--radius-sm)',
      },
      keyframes: {
        'slide-up': { from: { transform: 'translateY(100%)' }, to: { transform: 'translateY(0)' } },
      },
      animation: {
        'slide-up': 'slide-up 300ms ease-out',
      },
    },
  },
  plugins: [tailwindcssAnimate],
}
