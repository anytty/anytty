import { AlertCircle } from 'lucide-react'

export function PreviewNotice({ title, message }: { title: string; message: string }) {
  return (
    <div className="flex h-56 flex-col items-center justify-center gap-3 px-6 text-center">
      <AlertCircle className="h-7 w-7 text-zinc-400" />
      <h3 className="text-[16px] font-bold text-zinc-900">{title}</h3>
      <p className="max-w-sm text-[14px] leading-6 text-zinc-500">{message}</p>
    </div>
  )
}
