import { useMemo, useState, type CSSProperties, type FormEvent } from 'react'
import { Eye, EyeOff, LockKeyhole, SquareTerminal } from 'lucide-react'

type LoginCopy = {
  eyebrow: string
  title: string
  subtitle: string
  password: string
  placeholder: string
  show: string
  hide: string
  submit: string
  submitting: string
  invalid: string
  throttled: (seconds: number) => string
  httpsRequired: string
  unavailable: string
}

const englishCopy: LoginCopy = {
  eyebrow: 'LOCAL WEB ACCESS',
  title: 'AnyTTY Web',
  subtitle: 'This terminal workspace is password protected.',
  password: 'Access password',
  placeholder: 'Enter password',
  show: 'Show password',
  hide: 'Hide password',
  submit: 'Unlock terminal',
  submitting: 'Unlocking...',
  invalid: 'The password is incorrect.',
  throttled: (seconds) => `Too many attempts. Try again in ${seconds} seconds.`,
  httpsRequired: 'Public Web access requires HTTPS.',
  unavailable: 'AnyTTY Web could not verify this password. Try again.',
}

const chineseCopy: LoginCopy = {
  eyebrow: '本地 WEB 访问',
  title: 'AnyTTY Web',
  subtitle: '这个终端工作区已启用密码保护。',
  password: '访问密码',
  placeholder: '输入密码',
  show: '显示密码',
  hide: '隐藏密码',
  submit: '解锁终端',
  submitting: '正在解锁...',
  invalid: '密码不正确。',
  throttled: (seconds) => `尝试次数过多，请在 ${seconds} 秒后重试。`,
  httpsRequired: '公网 Web 访问必须使用 HTTPS。',
  unavailable: 'AnyTTY Web 暂时无法验证密码，请重试。',
}

export function LocalWebLogin({ initialAppThemeStyle, onAuthenticated }: { initialAppThemeStyle: CSSProperties; onAuthenticated: () => Promise<void> }) {
  const copy = useMemo(() => globalThis.navigator.language.toLowerCase().startsWith('zh') ? chineseCopy : englishCopy, [])
  const [password, setPassword] = useState('')
  const [visible, setVisible] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (submitting || password === '') return
    setSubmitting(true)
    setError(null)
    try {
      const response = await fetch('/api/auth/login', {
        method: 'POST',
        cache: 'no-store',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      })
      if (response.status === 401) {
        setError(copy.invalid)
        return
      }
      if (response.status === 429) {
        const retryAfter = Math.max(1, Number.parseInt(response.headers.get('Retry-After') ?? '60', 10) || 60)
        setError(copy.throttled(retryAfter))
        return
      }
      if (response.status === 426) {
        setError(copy.httpsRequired)
        return
      }
      if (!response.ok) {
        setError(copy.unavailable)
        return
      }
      setPassword('')
      await onAuthenticated()
    } catch {
      setError(copy.unavailable)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <main className="flex h-full min-h-[100dvh] w-full items-center justify-center bg-[var(--anytty-app-bg)] px-6 py-10 text-[var(--anytty-app-text)] antialiased" style={initialAppThemeStyle}>
      <section className="w-full max-w-[23rem]" aria-labelledby="local-web-login-title">
        <div className="mb-9 flex items-center gap-3">
          <span className="flex size-10 shrink-0 items-center justify-center rounded border border-[var(--anytty-app-line-strong)] bg-[var(--anytty-app-surface)]" aria-hidden="true">
            <SquareTerminal className="size-5" strokeWidth={1.8} />
          </span>
          <div className="min-w-0">
            <p className="text-[0.6875rem] font-semibold uppercase text-[var(--anytty-app-muted)]">{copy.eyebrow}</p>
            <h1 id="local-web-login-title" className="text-xl font-semibold">{copy.title}</h1>
          </div>
        </div>

        <div className="border-t border-[var(--anytty-app-line)] pt-7">
          <div className="mb-6 flex items-start gap-3">
            <LockKeyhole className="mt-0.5 size-4 shrink-0 text-[var(--anytty-app-success)]" aria-hidden="true" />
            <p className="text-sm leading-6 text-[var(--anytty-app-muted)]">{copy.subtitle}</p>
          </div>
          <form onSubmit={submit}>
            <label className="mb-2 block text-sm font-medium" htmlFor="local-web-password">{copy.password}</label>
            <div className="relative">
              <input
                id="local-web-password"
                className="h-12 w-full rounded border border-[var(--anytty-app-line-strong)] bg-[var(--anytty-app-surface)] px-3 pr-12 text-base text-[var(--anytty-app-text)] outline-none transition-colors placeholder:text-[var(--anytty-app-muted)] focus:border-[var(--anytty-app-text)] focus:ring-2 focus:ring-white/15"
                type={visible ? 'text' : 'password'}
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder={copy.placeholder}
                autoComplete="current-password"
                autoFocus
                spellCheck={false}
                aria-invalid={error ? 'true' : undefined}
                aria-describedby="local-web-login-error"
              />
              <button
                className="absolute right-0 top-0 flex size-12 items-center justify-center text-[var(--anytty-app-muted)] transition-colors hover:text-[var(--anytty-app-text)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-white/30"
                type="button"
                onClick={() => setVisible((current) => !current)}
                aria-label={visible ? copy.hide : copy.show}
                title={visible ? copy.hide : copy.show}
              >
                {visible ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
              </button>
            </div>
            <p id="local-web-login-error" className="min-h-10 pt-2 text-sm leading-5 text-red-400" role={error ? 'alert' : undefined}>{error}</p>
            <button
              className="flex h-12 w-full items-center justify-center rounded bg-[var(--anytty-app-text)] px-4 text-sm font-semibold text-[var(--anytty-app-bg)] transition-colors hover:bg-zinc-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--anytty-app-bg)] disabled:cursor-not-allowed disabled:opacity-50"
              type="submit"
              disabled={submitting || password === ''}
            >
              {submitting ? copy.submitting : copy.submit}
            </button>
          </form>
        </div>
      </section>
    </main>
  )
}
