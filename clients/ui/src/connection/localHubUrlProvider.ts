export interface LocalHubUrlProvider {
  getLocalHubUrl(): Promise<string | null>
}

export class ManualLocalHubUrlProvider implements LocalHubUrlProvider {
  constructor(private readonly url: string) {}

  async getLocalHubUrl(): Promise<string | null> {
    const trimmed = this.url.trim()
    return trimmed === '' ? null : trimmed
  }
}

export class QRLocalHubUrlProvider implements LocalHubUrlProvider {
  constructor(private readonly readUrl: () => Promise<string | null> | string | null) {}

  async getLocalHubUrl(): Promise<string | null> {
    const value = await this.readUrl()
    const trimmed = value?.trim() ?? ''
    return trimmed === '' ? null : trimmed
  }
}
