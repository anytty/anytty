import { NativeConnection, type NativeSessionDemandResult } from './plugins/nativeConnection'

export type NativeSessionDemandInput = { endpointIds: string[] }
export type NativeSessionDemandSink = (input: NativeSessionDemandInput) => Promise<NativeSessionDemandResult | void>

/** Owns the current renderer's complete native connection demand projection. */
export class NativeSessionDemandCoordinator {
  private desiredEndpointIds = new Set<string>()
  private queue: Promise<void> = Promise.resolve()
  private revision = 0
  private commandGeneration = 0
  private goManagedEndpointIds = new Set<string>()

  constructor(
    private readonly replaceDemand: NativeSessionDemandSink = (input) => NativeConnection.replaceSessionDemand(input),
  ) {}

  reconcileRenderer(): Promise<void> {
    return this.enqueueSnapshot(this.snapshot())
  }

  clearForUserStop(): Promise<void> {
    this.desiredEndpointIds.clear()
    this.goManagedEndpointIds.clear()
    this.revision += 1
    this.commandGeneration += 1
    return this.enqueueSnapshot(this.snapshot())
  }

  setActive(machineId: string, active: boolean): Promise<void> {
    const endpointId = machineId.trim()
    if (!endpointId) return Promise.reject(new Error('machineId is required'))
    const alreadyActive = this.desiredEndpointIds.has(endpointId)
    if (alreadyActive === active) return this.reconcileRenderer()

    const previous = new Set(this.desiredEndpointIds)
    if (active) this.desiredEndpointIds.add(endpointId)
    else this.desiredEndpointIds.delete(endpointId)
    const revision = ++this.revision
    const update = this.enqueueSnapshot(this.snapshot())
    return update.catch((failure: unknown) => {
      if (this.revision === revision) this.desiredEndpointIds = previous
      throw failure
    })
  }

  isGoManaged(machineId: string): boolean {
    return this.goManagedEndpointIds.has(machineId.trim())
  }

  private snapshot(): string[] {
    return [...this.desiredEndpointIds].sort()
  }

  private enqueueSnapshot(endpointIds: string[]): Promise<void> {
    const commandGeneration = this.commandGeneration
    const operation = this.queue.then(async () => {
      if (commandGeneration !== this.commandGeneration) return
      const result = await this.replaceDemand({ endpointIds })
      if (commandGeneration !== this.commandGeneration) return
      this.goManagedEndpointIds = new Set(result?.goManagedEndpointIds ?? [])
    })
    this.queue = operation.catch(() => undefined)
    return operation
  }
}

export const nativeSessionDemand = new NativeSessionDemandCoordinator()
