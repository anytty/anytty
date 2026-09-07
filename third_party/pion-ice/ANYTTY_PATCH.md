# Network-Scoped Gathering

Source: github.com/pion/ice/v4 v4.2.1 (MIT; original source and tests retained).

The local change in gather.go recognizes an optional network extension:
`ICEGatherNetworks() []transport.Net`. STUN and TURN gathering run on these
network-bound transports instead of using a single wildcard/default socket.
Active TCP checks additionally use an optional `DialTCPContext` extension to
preserve the candidate's source network and cancellation deadline. Other
transports retain upstream behavior. ICE checks, candidate selection,
authentication and protocol handling remain upstream Pion implementations.

This is necessary because enumerating host interfaces alone does not make
upstream v4.2.1 gather public addresses or relay allocations on each network.
Remove the replacement when upstream supports network-scoped gathering.

## Relay-Only Default Wait

The functional-option constructor derives the default Relay nomination wait
after applying candidate-type options. Otherwise its initially mixed candidate
defaults leave a two-second wait even when WebRTC requests Relay-only gathering.
The legacy constructor already defaults Relay-only to zero. Explicit waits from
either API retain precedence, independent of option ordering. Connectivity
checks, nomination validation, DTLS, and application authorization are unchanged.
