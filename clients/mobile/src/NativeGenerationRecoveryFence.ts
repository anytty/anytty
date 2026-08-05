/** Ensures only the latest native network epoch may commit connection readiness. */
export class NativeGenerationRecoveryFence {
  private attempt = 0
  private nativeEpoch = -1
  private nativeAttempt = -1
  private nativeReadyClaimed = false
  private nativeFailed = false

  beginAttempt(): number {
    this.attempt += 1
    return this.attempt
  }

  invalidate(): void {
    this.attempt += 1
  }

  beginNativeEpoch(epoch: number): number | null {
    if (!Number.isSafeInteger(epoch) || epoch <= this.nativeEpoch) return null
    this.nativeEpoch = epoch
    this.nativeReadyClaimed = false
    this.nativeFailed = false
    this.nativeAttempt = this.beginAttempt()
    return this.nativeAttempt
  }

  claimNativeReadyAttempt(epoch: number): number | null {
    if (
      epoch !== this.nativeEpoch ||
      this.nativeAttempt !== this.attempt ||
      this.nativeReadyClaimed ||
      this.nativeFailed
    ) return null
    this.nativeReadyClaimed = true
    return this.nativeAttempt
  }

  failNativeEpoch(epoch: number): number | null {
    if (epoch !== this.nativeEpoch || this.nativeAttempt !== this.attempt || this.nativeFailed) return null
    this.nativeFailed = true
    return this.beginAttempt()
  }

  isCurrent(attempt: number): boolean {
    return attempt === this.attempt
  }
}
