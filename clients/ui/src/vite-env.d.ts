declare module '*?raw' {
  const content: string
  export default content
}

declare module '*.png' {
  const url: string
  export default url
}

declare module '*.svg' {
  const url: string
  export default url
}

interface ImportMeta {
  glob<T = unknown>(
    pattern: string,
    options?: {
      eager?: boolean
      import?: string
      query?: string
    },
  ): Record<string, T>
}
