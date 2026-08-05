package core

import (
	"context"
	"sync"
	"time"
)

const terminalOutputChunkBytes = 16 << 10

type terminalOutputConsumer uint8

const (
	terminalOutputConsumerLive terminalOutputConsumer = iota
	terminalOutputConsumerHistory
	terminalOutputConsumerCount
)

func (consumer terminalOutputConsumer) bit() uint8 { return 1 << consumer }

func (consumer terminalOutputConsumer) String() string {
	if consumer == terminalOutputConsumerHistory {
		return "history"
	}
	return "live"
}

type terminalOutputGap struct {
	Consumer     terminalOutputConsumer
	Epoch        uint64
	DroppedBytes uint64
	throughSeq   uint64
}

type terminalOutputNode struct {
	payload []byte
	seq     uint64
	pending uint8
	next    *terminalOutputNode
}

type terminalOutputConsumerState struct {
	active          bool
	started         bool
	done            chan struct{}
	doneOnce        sync.Once
	next            *terminalOutputNode
	inFlight        *terminalOutputNode
	inFlightLast    *terminalOutputNode
	gapInFlight     *terminalOutputGap
	completedSeq    uint64
	pendingGapBytes uint64
	pendingGapSeq   uint64
	epoch           uint64
	droppedBytes    uint64
	gapCount        uint64
	unavailable     error
	failurePending  bool
}

type terminalOutputBatch struct {
	first   *terminalOutputNode
	last    *terminalOutputNode
	payload []byte
}

type terminalOutputBufferStatus struct {
	Policy                 TerminalOutputOverflowPolicy
	CapacityBytes          int64
	ResidentBytes          int64
	AggregateResidentBytes int64
	AggregateBudgetBytes   int64
	WaitNanos              int64
	DroppedBytes           uint64
	GapCount               uint64
	Epoch                  uint64
	PendingGapBytes        uint64
	Unavailable            bool
	UnavailableReason      string
	Closed                 bool
}

// terminalOutputResidentBudget accounts only payload bytes retained by actual
// buffer nodes. It does not reserve every terminal's configured capacity.
type terminalOutputResidentBudget struct {
	mu       sync.Mutex
	cond     *sync.Cond
	maxBytes int64
	resident int64
	closed   bool
	waiters  int
	buffers  map[*terminalOutputBuffer]struct{}
}

func newTerminalOutputResidentBudget(maxBytes int64) *terminalOutputResidentBudget {
	if maxBytes < MinTerminalOutputResidentBudgetBytes || maxBytes > MaxTerminalOutputResidentBudgetBytes {
		maxBytes = DefaultTerminalOutputResidentBudgetBytes
	}
	budget := &terminalOutputResidentBudget{maxBytes: maxBytes, buffers: make(map[*terminalOutputBuffer]struct{})}
	budget.cond = sync.NewCond(&budget.mu)
	return budget
}

func (budget *terminalOutputResidentBudget) register(buffer *terminalOutputBuffer) {
	if budget == nil {
		return
	}
	budget.mu.Lock()
	if !budget.closed {
		budget.buffers[buffer] = struct{}{}
	}
	budget.mu.Unlock()
}

func (budget *terminalOutputResidentBudget) unregister(buffer *terminalOutputBuffer) {
	if budget == nil {
		return
	}
	budget.mu.Lock()
	delete(budget.buffers, buffer)
	budget.cond.Broadcast()
	budget.mu.Unlock()
}

// reserve waits only for block-policy buffers. A drop-policy caller makes one
// eviction pass across drop buffers and reports failure immediately if block
// buffers still pin the aggregate budget.
func (budget *terminalOutputResidentBudget) reserve(buffer *terminalOutputBuffer, bytes int64) bool {
	if budget == nil || bytes <= 0 {
		return true
	}
	waitStarted := time.Time{}
	defer func() {
		if !waitStarted.IsZero() {
			buffer.addWaitDuration(time.Since(waitStarted))
		}
	}()
	for {
		if !buffer.acceptingOutput() {
			return false
		}
		budget.mu.Lock()
		if budget.closed {
			budget.mu.Unlock()
			return false
		}
		if budget.resident+bytes <= budget.maxBytes {
			budget.resident += bytes
			budget.mu.Unlock()
			return true
		}
		if buffer.config.Overflow == TerminalOutputOverflowBlock {
			if waitStarted.IsZero() {
				waitStarted = time.Now()
			}
			budget.waiters++
			budget.cond.Wait()
			budget.waiters--
			budget.mu.Unlock()
			continue
		}
		shortage := budget.resident + bytes - budget.maxBytes
		candidates := make([]*terminalOutputBuffer, 0, len(budget.buffers))
		for candidate := range budget.buffers {
			if candidate.config.Overflow == TerminalOutputOverflowDrop {
				candidates = append(candidates, candidate)
			}
		}
		budget.mu.Unlock()

		remaining := shortage
		for _, candidate := range candidates {
			remaining -= candidate.dropForAggregate(remaining)
			if remaining <= 0 {
				break
			}
		}
		budget.mu.Lock()
		if !budget.closed && budget.resident+bytes <= budget.maxBytes {
			budget.resident += bytes
			budget.mu.Unlock()
			return true
		}
		budget.mu.Unlock()
		return false
	}
}

func (budget *terminalOutputResidentBudget) release(bytes int64) {
	if budget == nil || bytes <= 0 {
		return
	}
	budget.mu.Lock()
	budget.resident -= bytes
	if budget.resident < 0 {
		budget.resident = 0
	}
	budget.cond.Broadcast()
	budget.mu.Unlock()
}

func (budget *terminalOutputResidentBudget) wake() {
	if budget == nil {
		return
	}
	budget.mu.Lock()
	budget.cond.Broadcast()
	budget.mu.Unlock()
}

func (budget *terminalOutputResidentBudget) status() (int64, int64) {
	if budget == nil {
		return 0, 0
	}
	budget.mu.Lock()
	defer budget.mu.Unlock()
	return budget.resident, budget.maxBytes
}

func (budget *terminalOutputResidentBudget) close() {
	if budget == nil {
		return
	}
	budget.mu.Lock()
	budget.closed = true
	budget.cond.Broadcast()
	budget.mu.Unlock()
}

// terminalOutputBuffer is a concrete two-consumer terminal primitive. Each PTY
// payload node owns one byte slice regardless of whether history is enabled.
type terminalOutputBuffer struct {
	writeMu      sync.Mutex
	mu           sync.Mutex
	cond         *sync.Cond
	config       TerminalOutputBufferConfig
	budget       *terminalOutputResidentBudget
	head         *terminalOutputNode
	tail         *terminalOutputNode
	resident     int64
	nextSeq      uint64
	closed       bool
	closedCh     chan struct{}
	sealed       bool
	waitNanos    int64
	localWaiters int
	flushWaiters [terminalOutputConsumerCount]int
	consumers    [terminalOutputConsumerCount]terminalOutputConsumerState
}

func newTerminalOutputBuffer(config TerminalOutputBufferConfig, budget *terminalOutputResidentBudget, historyEnabled bool) *terminalOutputBuffer {
	buffer := &terminalOutputBuffer{config: config.normalized(), budget: budget, closedCh: make(chan struct{})}
	buffer.cond = sync.NewCond(&buffer.mu)
	buffer.consumers[terminalOutputConsumerLive] = terminalOutputConsumerState{active: true, done: make(chan struct{})}
	buffer.consumers[terminalOutputConsumerHistory] = terminalOutputConsumerState{active: historyEnabled, done: make(chan struct{})}
	if !historyEnabled {
		buffer.finishConsumerLocked(terminalOutputConsumerHistory)
	}
	budget.register(buffer)
	return buffer
}

// Write serializes writers through reservation and append. Consumer progress can
// only reduce resident bytes while writeMu is held, so the local capacity check
// remains valid after an aggregate reservation succeeds.
func (buffer *terminalOutputBuffer) Write(payload []byte) bool {
	buffer.writeMu.Lock()
	defer buffer.writeMu.Unlock()
	for len(payload) > 0 {
		size := len(payload)
		if size > terminalOutputChunkBytes {
			size = terminalOutputChunkBytes
		}
		if int64(size) > buffer.config.CapacityBytes {
			size = int(buffer.config.CapacityBytes)
		}
		if !buffer.writeChunk(payload[:size]) {
			return false
		}
		payload = payload[size:]
	}
	return true
}

func (buffer *terminalOutputBuffer) writeChunk(payload []byte) bool {
	bytes := int64(len(payload))
	started := time.Time{}
	buffer.mu.Lock()
	for !buffer.closed && !buffer.sealed && buffer.hasActiveConsumerLocked() && buffer.resident+bytes > buffer.config.CapacityBytes {
		if buffer.config.Overflow == TerminalOutputOverflowDrop {
			freed := buffer.dropOldestLocked(bytes)
			if freed == 0 {
				freed = buffer.dropIncomingLocked(uint64(len(payload)))
				buffer.cond.Broadcast()
				buffer.mu.Unlock()
				buffer.budget.release(freed)
				return true
			}
			buffer.mu.Unlock()
			buffer.budget.release(freed)
			buffer.mu.Lock()
			continue
		}
		if started.IsZero() {
			started = time.Now()
		}
		buffer.localWaiters++
		buffer.cond.Wait()
		buffer.localWaiters--
	}
	if !started.IsZero() {
		buffer.waitNanos += time.Since(started).Nanoseconds()
	}
	if buffer.closed || buffer.sealed {
		buffer.mu.Unlock()
		return false
	}
	if !buffer.hasActiveConsumerLocked() {
		buffer.mu.Unlock()
		return true
	}
	buffer.mu.Unlock()

	if !buffer.budget.reserve(buffer, bytes) {
		buffer.mu.Lock()
		if buffer.closed || buffer.sealed || buffer.config.Overflow == TerminalOutputOverflowBlock {
			buffer.mu.Unlock()
			return false
		}
		freed := buffer.dropIncomingLocked(uint64(len(payload)))
		buffer.cond.Broadcast()
		buffer.mu.Unlock()
		buffer.budget.release(freed)
		return true
	}

	buffer.mu.Lock()
	if buffer.closed || buffer.sealed {
		buffer.mu.Unlock()
		buffer.budget.release(bytes)
		return false
	}
	buffer.nextSeq++
	seq := buffer.nextSeq
	mask := uint8(0)
	freed := int64(0)
	for consumer := terminalOutputConsumer(0); consumer < terminalOutputConsumerCount; consumer++ {
		state := &buffer.consumers[consumer]
		if !state.active {
			continue
		}
		if state.pendingGapBytes > 0 || state.gapInFlight != nil {
			buffer.dropQueuedForConsumerLocked(consumer, state)
			buffer.addGapLocked(state, seq, uint64(len(payload)))
			continue
		}
		mask |= consumer.bit()
	}
	freed += buffer.reclaimLocked()
	if mask == 0 {
		buffer.cond.Broadcast()
		buffer.mu.Unlock()
		buffer.budget.release(bytes + freed)
		return true
	}
	node := &terminalOutputNode{payload: append([]byte(nil), payload...), seq: seq, pending: mask}
	if buffer.tail == nil {
		buffer.head = node
	} else {
		buffer.tail.next = node
	}
	buffer.tail = node
	buffer.resident += bytes
	for consumer := terminalOutputConsumer(0); consumer < terminalOutputConsumerCount; consumer++ {
		state := &buffer.consumers[consumer]
		if node.pending&consumer.bit() != 0 && state.next == nil && state.inFlight == nil {
			state.next = node
		}
	}
	buffer.cond.Broadcast()
	buffer.mu.Unlock()
	buffer.budget.release(freed)
	return true
}

func (buffer *terminalOutputBuffer) Run(consumer terminalOutputConsumer, ingest func([]byte) error, gap func(terminalOutputGap) error) {
	buffer.run(consumer, ingest, gap, nil)
}

type terminalOutputConsumerFailure struct {
	Err       error
	DuringGap bool
}

func (buffer *terminalOutputBuffer) run(consumer terminalOutputConsumer, ingest func([]byte) error, gap func(terminalOutputGap) error, failed func(terminalOutputConsumerFailure)) {
	buffer.mu.Lock()
	state := &buffer.consumers[consumer]
	if state.started {
		buffer.mu.Unlock()
		return
	}
	if !state.active || buffer.closed {
		buffer.finishConsumerLocked(consumer)
		buffer.mu.Unlock()
		return
	}
	state.started = true
	buffer.mu.Unlock()
	defer buffer.finishConsumer(consumer)

	for {
		batch, gapEvent, ok := buffer.next(consumer)
		if !ok {
			return
		}
		var err error
		if gapEvent != nil {
			if gap != nil {
				err = gap(*gapEvent)
			}
			buffer.completeGap(consumer, gapEvent, err, failed != nil)
		} else {
			if ingest != nil {
				err = ingest(batch.payload)
			}
			buffer.completeBatch(consumer, batch, err, failed != nil)
		}
		if err != nil {
			if failed != nil {
				failed(terminalOutputConsumerFailure{Err: buffer.ConsumerError(consumer), DuringGap: gapEvent != nil})
				buffer.completeFailureHandling(consumer)
			}
			return
		}
	}
}

func (buffer *terminalOutputBuffer) next(consumer terminalOutputConsumer) (*terminalOutputBatch, *terminalOutputGap, bool) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	state := &buffer.consumers[consumer]
	for state.active && state.pendingGapBytes == 0 && state.next == nil && !buffer.sealed && !buffer.closed {
		buffer.cond.Wait()
	}
	if !state.active || buffer.closed {
		return nil, nil, false
	}
	if state.pendingGapBytes > 0 {
		gap := &terminalOutputGap{
			Consumer: consumer, Epoch: state.epoch + 1,
			DroppedBytes: state.pendingGapBytes, throughSeq: state.pendingGapSeq,
		}
		state.pendingGapBytes = 0
		state.pendingGapSeq = 0
		state.gapInFlight = gap
		return nil, gap, true
	}
	if state.next == nil {
		return nil, nil, false
	}
	batch := &terminalOutputBatch{first: state.next, last: state.next, payload: state.next.payload}
	total := len(batch.payload)
	for node := buffer.nextPendingNodeLocked(consumer, state.next.next); node != nil && total+len(node.payload) <= terminalOutputChunkBytes; node = buffer.nextPendingNodeLocked(consumer, node.next) {
		batch.last = node
		total += len(node.payload)
	}
	if batch.last != batch.first {
		batch.payload = make([]byte, 0, total)
		for node := batch.first; node != nil; node = buffer.nextPendingNodeLocked(consumer, node.next) {
			batch.payload = append(batch.payload, node.payload...)
			if node == batch.last {
				break
			}
		}
	}
	state.inFlight = batch.first
	state.inFlightLast = batch.last
	return batch, nil, true
}

func (buffer *terminalOutputBuffer) completeBatch(consumer terminalOutputConsumer, batch *terminalOutputBatch, ingestErr error, failurePending bool) {
	buffer.mu.Lock()
	state := &buffer.consumers[consumer]
	if batch == nil || state.inFlight != batch.first || state.inFlightLast != batch.last {
		buffer.mu.Unlock()
		return
	}
	state.inFlight = nil
	state.inFlightLast = nil
	for node := batch.first; node != nil; node = buffer.nextPendingNodeLocked(consumer, node.next) {
		if node.pending&consumer.bit() != 0 {
			node.pending &^= consumer.bit()
			if node.seq > state.completedSeq {
				state.completedSeq = node.seq
			}
		}
		if node == batch.last {
			break
		}
	}
	state.next = buffer.nextPendingNodeLocked(consumer, batch.last.next)
	if ingestErr != nil {
		dropped := uint64(len(batch.payload))
		state.droppedBytes += uint64(len(batch.payload))
		if state.pendingGapBytes == 0 {
			state.gapCount++
		}
		dropped += state.pendingGapBytes
		state.pendingGapBytes = 0
		state.pendingGapSeq = 0
		released := buffer.releaseConsumerLocked(consumer, state.next)
		dropped += released
		state.droppedBytes += released
		state.unavailable = &TerminalOutputError{Consumer: consumer.String(), Epoch: state.epoch + 1, DroppedBytes: dropped, Cause: ingestErr}
		state.failurePending = failurePending
		state.active = false
		state.next = nil
	}
	freed := buffer.reclaimLocked()
	buffer.cond.Broadcast()
	buffer.mu.Unlock()
	buffer.budget.release(freed)
	if ingestErr != nil {
		buffer.budget.wake()
	}
}

func (buffer *terminalOutputBuffer) completeGap(consumer terminalOutputConsumer, gap *terminalOutputGap, gapErr error, failurePending bool) {
	buffer.mu.Lock()
	state := &buffer.consumers[consumer]
	if state.gapInFlight != gap {
		buffer.mu.Unlock()
		return
	}
	state.gapInFlight = nil
	if gapErr != nil {
		dropped := gap.DroppedBytes + state.pendingGapBytes
		state.pendingGapBytes = 0
		state.pendingGapSeq = 0
		released := buffer.releaseConsumerLocked(consumer, state.next)
		dropped += released
		state.droppedBytes += released
		state.unavailable = &TerminalOutputError{
			Consumer: consumer.String(), Epoch: gap.Epoch,
			DroppedBytes: dropped, Cause: gapErr,
		}
		state.failurePending = failurePending
		state.active = false
		state.next = nil
	} else {
		state.epoch = gap.Epoch
		if gap.throughSeq > state.completedSeq {
			state.completedSeq = gap.throughSeq
		}
	}
	freed := buffer.reclaimLocked()
	buffer.cond.Broadcast()
	buffer.mu.Unlock()
	buffer.budget.release(freed)
	if gapErr != nil {
		buffer.budget.wake()
	}
}

func (buffer *terminalOutputBuffer) completeFailureHandling(consumer terminalOutputConsumer) {
	buffer.mu.Lock()
	buffer.consumers[consumer].failurePending = false
	buffer.cond.Broadcast()
	buffer.mu.Unlock()
}

func (buffer *terminalOutputBuffer) dropForAggregate(required int64) int64 {
	if required <= 0 || buffer.config.Overflow != TerminalOutputOverflowDrop {
		return 0
	}
	buffer.mu.Lock()
	freed := buffer.dropOldestLocked(required)
	buffer.cond.Broadcast()
	buffer.mu.Unlock()
	buffer.budget.release(freed)
	return freed
}

func (buffer *terminalOutputBuffer) dropOldestLocked(required int64) int64 {
	if required <= 0 {
		return 0
	}
	before := buffer.resident
	potential := int64(0)
	for node := buffer.head; node != nil && potential < required; node = node.next {
		for consumer := terminalOutputConsumer(0); consumer < terminalOutputConsumerCount; consumer++ {
			state := &buffer.consumers[consumer]
			if node.pending&consumer.bit() == 0 || outputNodeInFlightLocked(state, node) {
				continue
			}
			node.pending &^= consumer.bit()
			buffer.addGapLocked(state, node.seq, uint64(len(node.payload)))
			if state.next == node {
				state.next = buffer.nextPendingNodeLocked(consumer, node.next)
			}
		}
		if node.pending == 0 {
			potential += int64(len(node.payload))
		}
	}
	buffer.reclaimLocked()
	return before - buffer.resident
}

// dropIncomingLocked turns an unreservable input chunk into an ordered gap. Any
// queued payload for that consumer is conservatively included so the gap is
// always delivered before the next payload assigned to its new parser epoch.
func (buffer *terminalOutputBuffer) dropIncomingLocked(bytes uint64) int64 {
	buffer.nextSeq++
	seq := buffer.nextSeq
	for consumer := terminalOutputConsumer(0); consumer < terminalOutputConsumerCount; consumer++ {
		state := &buffer.consumers[consumer]
		if !state.active {
			continue
		}
		for node := buffer.head; node != nil; node = node.next {
			if outputNodeInFlightLocked(state, node) || node.pending&consumer.bit() == 0 {
				continue
			}
			node.pending &^= consumer.bit()
			buffer.addGapLocked(state, node.seq, uint64(len(node.payload)))
		}
		state.next = nil
		buffer.addGapLocked(state, seq, bytes)
	}
	return buffer.reclaimLocked()
}

func (buffer *terminalOutputBuffer) addGapLocked(state *terminalOutputConsumerState, throughSeq uint64, bytes uint64) {
	if bytes == 0 {
		return
	}
	if state.pendingGapBytes == 0 {
		state.gapCount++
	}
	state.pendingGapBytes += bytes
	if throughSeq > state.pendingGapSeq {
		state.pendingGapSeq = throughSeq
	}
	state.droppedBytes += bytes
}

func (buffer *terminalOutputBuffer) nextPendingNodeLocked(consumer terminalOutputConsumer, from *terminalOutputNode) *terminalOutputNode {
	for node := from; node != nil; node = node.next {
		if node.pending&consumer.bit() != 0 {
			return node
		}
	}
	return nil
}

func (buffer *terminalOutputBuffer) releaseConsumerLocked(consumer terminalOutputConsumer, from *terminalOutputNode) uint64 {
	released := uint64(0)
	for node := from; node != nil; node = node.next {
		if node.pending&consumer.bit() != 0 {
			node.pending &^= consumer.bit()
			released += uint64(len(node.payload))
		}
	}
	return released
}

func (buffer *terminalOutputBuffer) dropQueuedForConsumerLocked(consumer terminalOutputConsumer, state *terminalOutputConsumerState) {
	for node := buffer.head; node != nil; node = node.next {
		if outputNodeInFlightLocked(state, node) || node.pending&consumer.bit() == 0 {
			continue
		}
		node.pending &^= consumer.bit()
		buffer.addGapLocked(state, node.seq, uint64(len(node.payload)))
	}
	if state.inFlight == nil {
		state.next = nil
	}
}

func outputNodeInFlightLocked(state *terminalOutputConsumerState, target *terminalOutputNode) bool {
	if state == nil || target == nil || state.inFlight == nil {
		return false
	}
	for node := state.inFlight; node != nil; node = node.next {
		if node == target {
			return true
		}
		if node == state.inFlightLast {
			return false
		}
	}
	return false
}

func (buffer *terminalOutputBuffer) reclaimLocked() int64 {
	freed := int64(0)
	var previous *terminalOutputNode
	for node := buffer.head; node != nil; {
		next := node.next
		if node.pending != 0 {
			previous = node
			node = next
			continue
		}
		if previous == nil {
			buffer.head = next
		} else {
			previous.next = next
		}
		if buffer.tail == node {
			buffer.tail = previous
		}
		freed += int64(len(node.payload))
		buffer.resident -= int64(len(node.payload))
		node.payload = nil
		node.next = nil
		node = next
	}
	return freed
}

func (buffer *terminalOutputBuffer) Flush(ctx context.Context, consumer terminalOutputConsumer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	stopCancelNotify := context.AfterFunc(ctx, func() {
		buffer.mu.Lock()
		buffer.cond.Broadcast()
		buffer.mu.Unlock()
	})
	defer stopCancelNotify()

	buffer.mu.Lock()
	target := buffer.nextSeq
	for {
		state := &buffer.consumers[consumer]
		if state.unavailable != nil {
			if !state.failurePending {
				err := state.unavailable
				buffer.mu.Unlock()
				return err
			}
		} else if !state.active || state.completedSeq >= target {
			buffer.mu.Unlock()
			return nil
		}
		if err := ctx.Err(); err != nil {
			buffer.mu.Unlock()
			return err
		}
		buffer.flushWaiters[consumer]++
		buffer.cond.Wait()
		buffer.flushWaiters[consumer]--
	}
}

func (buffer *terminalOutputBuffer) Seal() {
	buffer.mu.Lock()
	buffer.sealed = true
	buffer.cond.Broadcast()
	buffer.mu.Unlock()
	buffer.budget.wake()
}

func (buffer *terminalOutputBuffer) Close() {
	buffer.mu.Lock()
	if buffer.closed {
		buffer.mu.Unlock()
		return
	}
	buffer.closed = true
	close(buffer.closedCh)
	for consumer := terminalOutputConsumer(0); consumer < terminalOutputConsumerCount; consumer++ {
		state := &buffer.consumers[consumer]
		if state.active {
			for node := buffer.head; node != nil; node = node.next {
				if outputNodeInFlightLocked(state, node) || node.pending&consumer.bit() == 0 {
					continue
				}
				node.pending &^= consumer.bit()
				buffer.addGapLocked(state, node.seq, uint64(len(node.payload)))
			}
			state.next = nil
		}
		state.active = false
		if !state.started {
			buffer.finishConsumerLocked(consumer)
		}
	}
	freed := buffer.reclaimLocked()
	buffer.cond.Broadcast()
	buffer.mu.Unlock()
	buffer.budget.release(freed)
	buffer.budget.unregister(buffer)
}

func (buffer *terminalOutputBuffer) Closed() <-chan struct{} {
	return buffer.closedCh
}

func (buffer *terminalOutputBuffer) Wait() {
	for consumer := terminalOutputConsumer(0); consumer < terminalOutputConsumerCount; consumer++ {
		<-buffer.consumers[consumer].done
	}
}

func (buffer *terminalOutputBuffer) Status(consumer terminalOutputConsumer) terminalOutputBufferStatus {
	if buffer == nil {
		return terminalOutputBufferStatus{}
	}
	buffer.mu.Lock()
	state := &buffer.consumers[consumer]
	status := terminalOutputBufferStatus{
		Policy: buffer.config.Overflow, CapacityBytes: buffer.config.CapacityBytes,
		ResidentBytes: buffer.resident, WaitNanos: buffer.waitNanos,
		DroppedBytes: state.droppedBytes, GapCount: state.gapCount,
		Epoch: state.epoch, PendingGapBytes: state.pendingGapBytes, Closed: buffer.closed,
	}
	if state.unavailable != nil {
		status.Unavailable = true
		status.UnavailableReason = state.unavailable.Error()
	}
	buffer.mu.Unlock()
	status.AggregateResidentBytes, status.AggregateBudgetBytes = buffer.budget.status()
	return status
}

func (buffer *terminalOutputBuffer) ConsumerError(consumer terminalOutputConsumer) error {
	if buffer == nil {
		return nil
	}
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.consumers[consumer].unavailable
}

func (buffer *terminalOutputBuffer) acceptingOutput() bool {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return !buffer.closed && !buffer.sealed && buffer.hasActiveConsumerLocked()
}

func (buffer *terminalOutputBuffer) addWaitDuration(duration time.Duration) {
	buffer.mu.Lock()
	buffer.waitNanos += duration.Nanoseconds()
	buffer.mu.Unlock()
}

func (buffer *terminalOutputBuffer) hasActiveConsumerLocked() bool {
	for consumer := terminalOutputConsumer(0); consumer < terminalOutputConsumerCount; consumer++ {
		if buffer.consumers[consumer].active {
			return true
		}
	}
	return false
}

func (buffer *terminalOutputBuffer) finishConsumer(consumer terminalOutputConsumer) {
	buffer.mu.Lock()
	buffer.finishConsumerLocked(consumer)
	buffer.mu.Unlock()
}

func (buffer *terminalOutputBuffer) finishConsumerLocked(consumer terminalOutputConsumer) {
	state := &buffer.consumers[consumer]
	state.doneOnce.Do(func() { close(state.done) })
}
