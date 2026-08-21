#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
baseline_ref="${ANYTTY_MEMORY_BENCH_BASELINE_REF:-f9eb081}"
terminal_count="${ANYTTY_MEMORY_BENCH_TERMINALS:-5}"
terminal_cols="${ANYTTY_MEMORY_BENCH_COLS:-157}"
terminal_rows="${ANYTTY_MEMORY_BENCH_ROWS:-79}"
settle_seconds="${ANYTTY_MEMORY_BENCH_SETTLE_SECONDS:-5}"
sample_count="${ANYTTY_MEMORY_BENCH_SAMPLES:-5}"
burst_bytes="${ANYTTY_MEMORY_BENCH_BURST_BYTES:-67108864}"
skip_burst="${ANYTTY_MEMORY_BENCH_SKIP_BURST:-0}"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
output_dir="${1:-$repo_root/.artifacts/single-vterm-memory/$timestamp}"
if [[ "$output_dir" != /* ]]; then
	output_dir="$repo_root/$output_dir"
fi

if [[ "$(uname -s)" != "Darwin" ]]; then
	echo "this benchmark requires macOS footprint(1)" >&2
	exit 2
fi
for value in "$terminal_count" "$terminal_cols" "$terminal_rows" "$settle_seconds" "$sample_count" "$burst_bytes"; do
	if ! [[ "$value" =~ ^[0-9]+$ ]] || [[ "$value" -eq 0 ]]; then
		echo "benchmark numeric settings must be positive integers" >&2
		exit 2
	fi
done
if [[ -e "$output_dir" ]]; then
	echo "benchmark output already exists: $output_dir" >&2
	exit 1
fi

mkdir -p "$output_dir/bin" "$output_dir/source"
daemon_pids=()
sampler_pid=""

stop_pid() {
	local pid="$1"
	if ! kill -0 "$pid" 2>/dev/null; then
		wait "$pid" 2>/dev/null || true
		return
	fi
	kill -TERM "$pid" 2>/dev/null || true
	local attempt
	for attempt in $(seq 1 100); do
		if ! kill -0 "$pid" 2>/dev/null; then
			wait "$pid" 2>/dev/null || true
			return
		fi
		sleep 0.05
	done
	kill -KILL "$pid" 2>/dev/null || true
	wait "$pid" 2>/dev/null || true
}

cleanup() {
	if [[ -n "$sampler_pid" ]]; then
		kill -TERM "$sampler_pid" 2>/dev/null || true
		wait "$sampler_pid" 2>/dev/null || true
	fi
	local pid
	for pid in ${daemon_pids[@]+"${daemon_pids[@]}"}; do
		stop_pid "$pid"
	done
}
trap cleanup EXIT INT TERM

baseline_tar="$output_dir/source/$baseline_ref.tar"
baseline_source="$output_dir/source/$baseline_ref"
git -C "$repo_root" archive --format=tar --output="$baseline_tar" "$baseline_ref"
mkdir -p "$baseline_source"
tar -xf "$baseline_tar" -C "$baseline_source"
rm "$baseline_tar"
install -m 0644 "$repo_root/cmd/anytty/memory_bench_signal_unix.go" "$baseline_source/cmd/anytty/memory_bench_signal_unix.go"

echo "building baseline $baseline_ref"
(
	cd "$baseline_source"
	GOWORK=off go build -tags anytty_memory_bench -trimpath -o "$output_dir/bin/anytty-baseline" ./cmd/anytty
)
echo "building current $(git -C "$repo_root" rev-parse --short HEAD)"
(
	cd "$repo_root"
	GOWORK=off go build -tags anytty_memory_bench -trimpath -o "$output_dir/bin/anytty-current" ./cmd/anytty
	GOWORK=off go build -tags 'anytty_memory_bench anytty_dev_commands' -trimpath -o "$output_dir/bin/anytty-bench-client" ./cmd/anytty
)

client="$output_dir/bin/anytty-bench-client"
idle_tsv="$output_dir/idle.tsv"
printf 'variant\tref\tpid\tterminals\tcols\trows\theap_alloc\theap_inuse\theap_sys\tphysical_median\tphysical_min\tphysical_max\tphysical_peak\trss_kb\tcpu_percent\tnum_gc\n' >"$idle_tsv"

wait_for_daemon() {
	local socket="$1"
	local run_dir="$2"
	local pid="$3"
	local attempt
	for attempt in $(seq 1 200); do
		if ! kill -0 "$pid" 2>/dev/null; then
			echo "daemon exited before readiness; see $run_dir/daemon.stdout" >&2
			return 1
		fi
		if [[ -S "$socket" ]] && env \
			XDG_CONFIG_HOME="$run_dir/config" \
			XDG_STATE_HOME="$run_dir/state" \
			XDG_RUNTIME_DIR="$run_dir/runtime" \
			"$client" --socket "$socket" --log-file "$run_dir/client.log" terminal list --no-header >/dev/null 2>&1; then
			return 0
		fi
		sleep 0.05
	done
	echo "daemon readiness timed out; see $run_dir/daemon.stdout" >&2
	return 1
}

start_daemon() {
	local variant="$1"
	local binary="$2"
	run_dir="$output_dir/$variant"
	socket="$run_dir/runtime/daemon.sock"
	mkdir -p "$run_dir/config" "$run_dir/state" "$run_dir/runtime" "$run_dir/memstats"
	chmod 700 "$run_dir/config" "$run_dir/state" "$run_dir/runtime" "$run_dir/memstats"
	env \
		XDG_CONFIG_HOME="$run_dir/config" \
		XDG_STATE_HOME="$run_dir/state" \
		XDG_RUNTIME_DIR="$run_dir/runtime" \
		ANYTTY_DAEMON_MEMSTATS_DIR="$run_dir/memstats" \
		ANYTTY_DIAG_STAGE_FILE="$run_dir/stage" \
		ANYTTY_DIRECT_SIGNALING_LISTEN="127.0.0.1:0" \
		ANYTTY_DIRECT_ICE_TCP_LISTEN="127.0.0.1:0" \
		"$binary" --socket "$socket" --log-file "$run_dir/daemon.log" daemon run >"$run_dir/daemon.stdout" 2>&1 &
	daemon_pid=$!
	daemon_pids+=("$daemon_pid")
	wait_for_daemon "$socket" "$run_dir" "$daemon_pid"
}

run_client() {
	local run_dir="$1"
	shift
	env \
		XDG_CONFIG_HOME="$run_dir/config" \
		XDG_STATE_HOME="$run_dir/state" \
		XDG_RUNTIME_DIR="$run_dir/runtime" \
		"$client" --socket "$run_dir/runtime/daemon.sock" --log-file "$run_dir/client.log" "$@"
}

create_idle_terminals() {
	local run_dir="$1"
	local index
	for index in $(seq 1 "$terminal_count"); do
		run_client "$run_dir" terminal create \
			--name "memory-bench-$index" \
			--cols "$terminal_cols" \
			--rows "$terminal_rows" \
			-- /bin/zsh -f >"$run_dir/create-$index.out"
	done
}

write_stage() {
	local run_dir="$1"
	local stage="$2"
	printf '%s\n' "$stage" >"$run_dir/stage"
}

force_gc() {
	local run_dir="$1"
	local pid="$2"
	local stage="$3"
	write_stage "$run_dir" "$stage"
	kill -USR1 "$pid"
	sleep "$settle_seconds"
}

sample_memstats() {
	local run_dir="$1"
	local pid="$2"
	local stage="$3"
	local file="$run_dir/memstats/memstats.tsv"
	local before=0
	if [[ -f "$file" ]]; then
		before="$(wc -l <"$file" | tr -d ' ')"
	fi
	write_stage "$run_dir" "$stage"
	kill -USR2 "$pid"
	local attempt lines=0
	for attempt in $(seq 1 100); do
		if [[ -f "$file" ]]; then
			lines="$(wc -l <"$file" | tr -d ' ')"
			if [[ "$lines" -gt "$before" ]]; then
				tail -n 1 "$file"
				return 0
			fi
		fi
		sleep 0.05
	done
	echo "memstats sample timed out for stage $stage" >&2
	return 1
}

sample_footprints() {
	local run_dir="$1"
	local pid="$2"
	local stage="$3"
	local values="$run_dir/$stage-physical.values"
	: >"$values"
	local index output physical
	for index in $(seq 1 "$sample_count"); do
		output="$run_dir/$stage-footprint-$index.txt"
		/usr/bin/footprint --pid "$pid" --format bytes --noCategories >"$output" 2>&1
		physical="$(awk '/phys_footprint:/ { print $2; exit }' "$output")"
		if ! [[ "$physical" =~ ^[0-9]+$ ]]; then
			echo "could not parse physical footprint from $output" >&2
			return 1
		fi
		printf '%s\n' "$physical" >>"$values"
		sleep 1
	done
	physical_min="$(sort -n "$values" | head -n 1)"
	physical_max="$(sort -n "$values" | tail -n 1)"
	physical_median="$(sort -n "$values" | awk '{ values[NR]=$1 } END { if (NR % 2) print values[(NR+1)/2]; else print int((values[NR/2]+values[NR/2+1])/2) }')"
	physical_peak="$(awk '/phys_footprint_peak:/ { print $2; exit }' "$run_dir/$stage-footprint-$sample_count.txt")"
}

measure_idle() {
	local variant="$1"
	local ref="$2"
	local binary="$3"
	start_daemon "$variant" "$binary"
	local active_run_dir="$run_dir"
	local active_pid="$daemon_pid"
	create_idle_terminals "$active_run_dir"
	force_gc "$active_run_dir" "$active_pid" idle_gc
	local memline
	memline="$(sample_memstats "$active_run_dir" "$active_pid" idle_stable)"
	sample_footprints "$active_run_dir" "$active_pid" idle_stable
	local heap_alloc heap_sys heap_inuse num_gc rss_kb cpu_percent
	heap_alloc="$(awk -F '\t' '{ print $5 }' <<<"$memline")"
	heap_sys="$(awk -F '\t' '{ print $6 }' <<<"$memline")"
	heap_inuse="$(awk -F '\t' '{ print $8 }' <<<"$memline")"
	num_gc="$(awk -F '\t' '{ print $19 }' <<<"$memline")"
	read -r rss_kb cpu_percent < <(ps -p "$active_pid" -o rss=,%cpu=)
	printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
		"$variant" "$ref" "$active_pid" "$terminal_count" "$terminal_cols" "$terminal_rows" \
		"$heap_alloc" "$heap_inuse" "$heap_sys" "$physical_median" "$physical_min" "$physical_max" \
		"$physical_peak" "$rss_kb" "$cpu_percent" "$num_gc" >>"$idle_tsv"
}

baseline_binary="$output_dir/bin/anytty-baseline"
current_binary="$output_dir/bin/anytty-current"
current_ref="$(git -C "$repo_root" rev-parse --short HEAD)"
if ! git -C "$repo_root" diff --quiet || ! git -C "$repo_root" diff --cached --quiet; then
	current_ref="$current_ref-dirty"
fi

echo "measuring baseline idle daemon"
measure_idle baseline "$baseline_ref" "$baseline_binary"
baseline_pid="$daemon_pid"
stop_pid "$baseline_pid"
daemon_pids=()

echo "measuring current idle daemon"
measure_idle current "$current_ref" "$current_binary"
current_pid="$daemon_pid"
current_run_dir="$run_dir"

baseline_physical="$(awk -F '\t' '$1 == "baseline" { print $10 }' "$idle_tsv")"
current_physical="$(awk -F '\t' '$1 == "current" { print $10 }' "$idle_tsv")"
reduction_percent="$(awk -v before="$baseline_physical" -v after="$current_physical" 'BEGIN { printf "%.2f", (before-after)*100/before }')"
{
	printf 'baseline_physical_bytes=%s\n' "$baseline_physical"
	printf 'current_physical_bytes=%s\n' "$current_physical"
	printf 'physical_reduction_percent=%s\n' "$reduction_percent"
} >"$output_dir/idle-summary.txt"

idle_pass="$(awk -v before="$baseline_physical" -v after="$current_physical" 'BEGIN { print (after <= 45*1024*1024 && after <= before*0.65) ? "yes" : "no" }')"
printf 'idle_acceptance=%s\n' "$idle_pass" >>"$output_dir/idle-summary.txt"

if [[ "$skip_burst" != "1" ]]; then
	echo "running current 64 MiB burst"
	burst_tsv="$output_dir/burst.tsv"
	printf 'bytes\tduration_seconds\tpeak_heap_alloc\tpeak_heap_inuse\tpeak_heap_sys\tstable_heap_alloc\tstable_heap_inuse\tstable_heap_sys\tstable_physical_median\tphysical_peak\tstable_num_gc_delta\tstable_cpu_percent\tdropped_bytes\tgap_count\tunavailable\tbegin_found\tend_found\n' >"$burst_tsv"
	write_stage "$current_run_dir" burst
	(
		while kill -0 "$current_pid" 2>/dev/null; do
			kill -USR2 "$current_pid" 2>/dev/null || exit 0
			sleep 1
		done
	) &
	sampler_pid=$!
	burst_start="$(date +%s)"
	burst_command="printf 'BURST-%s\\r\\n' BEGIN; /usr/bin/perl -e '\$n=$burst_bytes; \$r=(\"A\"x155).\"\\r\\n\"; while (\$n >= length(\$r)) { print \$r; \$n -= length(\$r) } print substr(\$r, 0, \$n);'; printf '\\r\\nBURST-%s\\r\\n' END"
	run_client "$current_run_dir" terminal send local:memory-bench-1 --literal "$burst_command" --enter >"$current_run_dir/burst-send.out"
	burst_capture="$current_run_dir/burst-capture.txt"
	end_found=no
	for attempt in $(seq 1 300); do
		if run_client "$current_run_dir" terminal capture local:memory-bench-1 --lines 40 --cols "$terminal_cols" --timeout 30s >"$burst_capture" 2>"$current_run_dir/burst-capture.err" && grep -q 'BURST-END' "$burst_capture"; then
			end_found=yes
			break
		fi
		sleep 1
	done
	burst_end="$(date +%s)"
	kill -TERM "$sampler_pid" 2>/dev/null || true
	wait "$sampler_pid" 2>/dev/null || true
	sampler_pid=""
	if [[ "$end_found" != "yes" ]]; then
		echo "64 MiB burst did not reach its completion marker" >&2
		exit 1
	fi
	begin_found=no
	if run_client "$current_run_dir" history search memory-bench-1 BURST-BEGIN --fixed-strings --max-count 1 --color never >"$current_run_dir/burst-begin-search.txt"; then
		begin_found=yes
	fi
	if ! run_client "$current_run_dir" history search memory-bench-1 BURST-END --fixed-strings --max-count 1 --color never >"$current_run_dir/burst-end-search.txt"; then
		end_found=no
	fi
	run_client "$current_run_dir" v3 history-backlog memory-bench-1 >"$current_run_dir/burst-backlog.tsv"
	backlog_line="$(tail -n 1 "$current_run_dir/burst-backlog.tsv")"
	dropped_bytes="$(awk -F '\t' '{ print $8 }' <<<"$backlog_line")"
	gap_count="$(awk -F '\t' '{ print $9 }' <<<"$backlog_line")"
	unavailable="$(awk -F '\t' '{ print $11 }' <<<"$backlog_line")"
	peak_heap_alloc="$(awk -F '\t' '$3 == "burst" && $5 > max { max=$5 } END { print max+0 }' "$current_run_dir/memstats/memstats.tsv")"
	peak_heap_sys="$(awk -F '\t' '$3 == "burst" && $6 > max { max=$6 } END { print max+0 }' "$current_run_dir/memstats/memstats.tsv")"
	peak_heap_inuse="$(awk -F '\t' '$3 == "burst" && $8 > max { max=$8 } END { print max+0 }' "$current_run_dir/memstats/memstats.tsv")"
	force_gc "$current_run_dir" "$current_pid" burst_stable_gc
	stable_one="$(sample_memstats "$current_run_dir" "$current_pid" burst_stable_one)"
	sleep 5
	stable_two="$(sample_memstats "$current_run_dir" "$current_pid" burst_stable_two)"
	sample_footprints "$current_run_dir" "$current_pid" burst_stable
	stable_heap_alloc="$(awk -F '\t' '{ print $5 }' <<<"$stable_two")"
	stable_heap_sys="$(awk -F '\t' '{ print $6 }' <<<"$stable_two")"
	stable_heap_inuse="$(awk -F '\t' '{ print $8 }' <<<"$stable_two")"
	stable_num_gc_one="$(awk -F '\t' '{ print $19 }' <<<"$stable_one")"
	stable_num_gc_two="$(awk -F '\t' '{ print $19 }' <<<"$stable_two")"
	stable_num_gc_delta=$((stable_num_gc_two - stable_num_gc_one))
	read -r _ stable_cpu_percent < <(ps -p "$current_pid" -o rss=,%cpu=)
	printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
		"$burst_bytes" "$((burst_end - burst_start))" "$peak_heap_alloc" "$peak_heap_inuse" "$peak_heap_sys" \
		"$stable_heap_alloc" "$stable_heap_inuse" "$stable_heap_sys" "$physical_median" "$physical_peak" \
		"$stable_num_gc_delta" "$stable_cpu_percent" "$dropped_bytes" "$gap_count" "$unavailable" "$begin_found" "$end_found" >>"$burst_tsv"
	if [[ "$dropped_bytes" != "0" || "$gap_count" != "0" || "$unavailable" != "false" || "$begin_found" != "yes" || "$end_found" != "yes" ]]; then
		echo "burst history verification failed; see $burst_tsv" >&2
		exit 1
	fi
fi

stop_pid "$current_pid"
daemon_pids=()

cat "$idle_tsv"
cat "$output_dir/idle-summary.txt"
if [[ -f "$output_dir/burst.tsv" ]]; then
	cat "$output_dir/burst.tsv"
fi
echo "artifacts=$output_dir"

if [[ "$idle_pass" != "yes" ]]; then
	echo "idle daemon did not meet <=45 MiB and >=35% reduction acceptance" >&2
	exit 1
fi
