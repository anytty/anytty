export interface ParsedCandidate {
  foundation?: string
  component?: string
  protocol?: string
  address?: string
  port?: number
  candidateType?: string
  tcpType?: string
  relayProtocol?: string
  relatedAddress?: string
}

export function parseCandidateLine(candidateLine: string | undefined): ParsedCandidate | null {
  const line = candidateLine?.trim()
  if (!line) return null
  const parts = line.replace(/^a=/, '').split(/\s+/)
  if (!parts[0]?.startsWith('candidate:')) return null
  const typeIndex = parts.findIndex((part) => part === 'typ')
  const raddrIndex = parts.findIndex((part) => part === 'raddr')
  const tcpTypeIndex = parts.findIndex((part) => part === 'tcptype')
  const relayProtocolIndex = parts.findIndex((part) => part === 'relayProtocol')
  const out: ParsedCandidate = {
    foundation: parts[0].slice('candidate:'.length),
  }
  if (parts[1]) out.component = parts[1]
  if (parts[2]) out.protocol = parts[2].toLowerCase()
  if (parts[4]) out.address = parts[4]
  const port = Number(parts[5])
  if (port) out.port = port
  const candidateType = typeIndex >= 0 ? parts[typeIndex + 1] : undefined
  const tcpType = tcpTypeIndex >= 0 ? parts[tcpTypeIndex + 1] : undefined
  const relayProtocol = relayProtocolIndex >= 0 ? parts[relayProtocolIndex + 1] : undefined
  const relatedAddress = raddrIndex >= 0 ? parts[raddrIndex + 1] : undefined
  if (candidateType) out.candidateType = candidateType
  if (tcpType) out.tcpType = tcpType
  if (relayProtocol) out.relayProtocol = relayProtocol
  if (relatedAddress) out.relatedAddress = relatedAddress
  return out
}

export function parseSDPCandidates(sdp: string): ParsedCandidate[] {
  if (!sdp) return []
  return sdp.split(/\r?\n/)
    .map((raw) => raw.trim())
    .filter((line) => line.startsWith('a=candidate:'))
    .map((line) => parseCandidateLine(line.slice('a='.length)))
    .filter((candidate): candidate is ParsedCandidate => Boolean(candidate))
}

function countBy(values: string[]): Record<string, number> {
  const counts: Record<string, number> = {}
  for (const value of values) counts[value] = (counts[value] ?? 0) + 1
  return counts
}

export function sdpCandidateSummary(sdp: string): Record<string, unknown> {
  const candidates = parseSDPCandidates(sdp)
  return {
    count: candidates.length,
    byType: countBy(candidates.map((candidate) => candidate.candidateType ?? 'unknown')),
    byProtocol: countBy(candidates.map((candidate) => candidate.protocol ?? 'unknown')),
    candidates: candidates.slice(0, 12),
    truncated: candidates.length > 12,
  }
}

export function iceCandidateInitSummary(candidates: RTCIceCandidateInit[]): Record<string, unknown> {
  const parsed = candidates
    .map((candidate) => parseCandidateLine(candidate.candidate))
    .filter((candidate): candidate is ParsedCandidate => Boolean(candidate))
  return {
    count: candidates.length,
    byType: countBy(parsed.map((candidate) => candidate.candidateType ?? 'unknown')),
    byProtocol: countBy(parsed.map((candidate) => candidate.protocol ?? 'unknown')),
    candidates: parsed.slice(0, 12),
    truncated: parsed.length > 12,
  }
}

export interface PeerSessionDescription {
  type: 'offer' | 'answer'
  sdp: string
}

export function normalizeRemoteDescription(description: PeerSessionDescription): { description: PeerSessionDescription; candidates: RTCIceCandidateInit[] } {
  const normalized = splitOutAnswerCandidates(description.sdp)
  return {
    description: {
      ...description,
      sdp: normalized.sdp,
    },
    candidates: normalized.candidates,
  }
}

export function splitOutAnswerCandidates(sdp: string): { sdp: string; candidates: RTCIceCandidateInit[] } {
  const sessionLines: string[] = []
  const mediaLines: string[] = []
  const candidates: RTCIceCandidateInit[] = []
  let sessionFingerprint = ''
  let currentMid = ''
  let currentMLineIndex = -1
  let inMedia = false
  let normalized = false
  for (const rawLine of sdp.split(/\r?\n/)) {
    const line = rawLine.trim()
    if (line.startsWith('m=')) {
      currentMLineIndex += 1
      inMedia = true
    }
    if (line.startsWith('a=mid:')) currentMid = line.slice('a=mid:'.length)
    if (line === 'a=end-of-candidates') continue
    if (!inMedia && line.startsWith('a=fingerprint:')) {
      sessionFingerprint = line
      normalized = true
      continue
    }
    if (line.startsWith('a=candidate:')) {
      normalized = true
      if (/^a=candidate:\S+\s+2\s+/i.test(line)) continue
      candidates.push({
        candidate: line.slice('a='.length).replace(/\s+ufrag\s+\S+$/i, ''),
        ...(currentMid ? { sdpMid: currentMid } : {}),
        ...(currentMLineIndex >= 0 ? { sdpMLineIndex: currentMLineIndex } : {}),
      })
      continue
    }
    if (inMedia) {
      mediaLines.push(line)
    } else {
      sessionLines.push(line)
    }
  }
  const media = normalized ? normalizeApplicationMediaLines(mediaLines, sessionFingerprint) : mediaLines
  const normalizedSDP = [...sessionLines, ...media].join('\r\n')
  return { sdp: normalized ? `${normalizedSDP}\r\n` : normalizedSDP, candidates }
}

function normalizeApplicationMediaLines(lines: string[], fallbackFingerprint = ''): string[] {
  const mLine = lines.find((line) => line.startsWith('m=')) ?? ''
  if (!mLine.includes('webrtc-datachannel')) return lines.filter((line) => line !== 'a=sendrecv')
  const byPrefix = (prefix: string) => lines.find((line) => line.startsWith(prefix))
  return [
    mLine,
    byPrefix('c='),
    byPrefix('a=ice-ufrag:'),
    byPrefix('a=ice-pwd:'),
    byPrefix('a=ice-options:') ?? 'a=ice-options:trickle',
    byPrefix('a=fingerprint:') ?? fallbackFingerprint,
    byPrefix('a=setup:'),
    byPrefix('a=mid:'),
    byPrefix('a=sctp-port:'),
    'a=max-message-size:262144',
  ].filter((line): line is string => Boolean(line))
}
