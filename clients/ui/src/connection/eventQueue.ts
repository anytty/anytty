import { assertMessageBoundary, type ConnectionMessage } from './connectionMessageReducer'

export interface QueuedConnectionEvent {
  id: string
  sequence: number
  message: ConnectionMessage
}

export interface ConnectionEventQueueOptions {
  maxSize: number
}

export class ConnectionEventQueue {
  readonly maxSize: number
  droppedDuplicateCount = 0
  droppedBackpressureCount = 0

  private readonly seenIds = new Set<string>()
  private readonly events: QueuedConnectionEvent[] = []

  constructor(options: ConnectionEventQueueOptions) {
    if (!Number.isInteger(options.maxSize) || options.maxSize <= 0) {
      throw new Error('maxSize must be a positive integer')
    }
    this.maxSize = options.maxSize
  }

  enqueue(event: QueuedConnectionEvent): boolean {
    assertQueuedEvent(event)
    assertMessageBoundary(event.message)

    if (this.seenIds.has(event.id)) {
      this.droppedDuplicateCount += 1
      return false
    }

    this.seenIds.add(event.id)
    this.events.push(event)
    this.events.sort((left, right) => left.sequence - right.sequence)
    this.applyBackpressure()
    return true
  }

  flush(): QueuedConnectionEvent[] {
    const flushed = [...this.events].sort((left, right) => left.sequence - right.sequence)
    this.events.length = 0
    return flushed
  }

  get size(): number {
    return this.events.length
  }

  private applyBackpressure(): void {
    while (this.events.length > this.maxSize) {
      const dropIndex = this.findDropCandidateIndex()
      const [dropped] = this.events.splice(dropIndex, 1)
      if (dropped) {
        this.droppedBackpressureCount += 1
      }
    }
  }

  private findDropCandidateIndex(): number {
    const connectionChatterIndex = this.events.findIndex((event) =>
      event.message.type === 'connection.disconnected' ||
      event.message.type === 'connection.failed' ||
      event.message.type === 'terminal.channelClosed' ||
      event.message.type === 'file.channelClosed',
    )
    if (connectionChatterIndex >= 0) return connectionChatterIndex

    const nonIntentIndex = this.events.findIndex((event) => !isUserIntentOrLifecycle(event.message))
    if (nonIntentIndex >= 0) return nonIntentIndex

    return 0
  }
}

function assertQueuedEvent(event: QueuedConnectionEvent): void {
  if (!event.id) {
    throw new Error('queued connection event id is required')
  }
  if (!Number.isFinite(event.sequence)) {
    throw new Error('queued connection event sequence must be finite')
  }
}

function isUserIntentOrLifecycle(message: ConnectionMessage): boolean {
  return message.type.startsWith('user.') ||
    message.type.startsWith('app.') ||
    message.type.startsWith('network.')
}
