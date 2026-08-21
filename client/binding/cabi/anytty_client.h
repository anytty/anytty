#ifndef ANYTTY_CLIENT_H
#define ANYTTY_CLIENT_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define ANYTTY_CLIENT_ABI_VERSION 5u

typedef uint64_t anytty_handle_t;

typedef enum anytty_status_v1 {
  ANYTTY_STATUS_OK = 0,
  ANYTTY_STATUS_INVALID_ARGUMENT = 1,
  ANYTTY_STATUS_INVALID_HANDLE = 2,
  ANYTTY_STATUS_CLOSED = 3,
  ANYTTY_STATUS_CAPACITY = 4,
  ANYTTY_STATUS_INTERNAL = 5
} anytty_status_v1;

/* Inputs are borrowed only for the duration of the call. Event output is
 * wrapper-owned memory identified by buffer_handle and must be released once
 * with anytty_buffer_free. No returned data pointer may reference Go memory. */
typedef struct anytty_buffer_v1 {
  anytty_handle_t buffer_handle;
  const uint8_t *data;
  size_t length;
} anytty_buffer_v1;

uint32_t anytty_client_abi_version(void);
anytty_status_v1 anytty_engine_create(anytty_handle_t *out_engine_handle);
anytty_status_v1 anytty_bridge_start(anytty_handle_t engine_handle, const uint8_t *token, size_t token_length, uint16_t *out_port);
anytty_status_v1 anytty_local_probe(const uint8_t *local_discovery_proto, size_t local_discovery_length, uint8_t *out_reachable);
anytty_status_v1 anytty_supervisor_replace_demand(anytty_handle_t engine_handle, const uint8_t *demand_proto, size_t demand_length);
anytty_status_v1 anytty_supervisor_signal(anytty_handle_t engine_handle, const uint8_t *signal_proto, size_t signal_length);
anytty_status_v1 anytty_supervisor_wait_ready(anytty_handle_t engine_handle, uint32_t timeout_millis);
anytty_status_v1 anytty_supervisor_snapshot(anytty_handle_t engine_handle, anytty_buffer_v1 *out_snapshot_proto);
anytty_status_v1 anytty_engine_open_session(anytty_handle_t engine_handle, const uint8_t *request_proto, size_t request_length, anytty_handle_t *out_operation_handle);
anytty_status_v1 anytty_engine_execute(anytty_handle_t engine_handle, anytty_handle_t session_handle, const uint8_t *command_proto, size_t command_length, anytty_handle_t *out_operation_handle);
anytty_status_v1 anytty_engine_open_resource_stream(anytty_handle_t engine_handle, anytty_handle_t session_handle, const uint8_t *request_proto, size_t request_length, anytty_handle_t *out_stream_handle);
anytty_status_v1 anytty_engine_send_resource_stream_frame(anytty_handle_t engine_handle, anytty_handle_t stream_handle, const uint8_t *frame_proto, size_t frame_length);
anytty_status_v1 anytty_engine_close_resource_stream(anytty_handle_t engine_handle, anytty_handle_t stream_handle);
anytty_status_v1 anytty_engine_command(anytty_handle_t engine_handle, const uint8_t *command_proto, size_t command_length, anytty_handle_t *out_operation_handle);
anytty_status_v1 anytty_engine_next_event(anytty_handle_t engine_handle, uint32_t timeout_millis, anytty_buffer_v1 *out_event_proto);
anytty_status_v1 anytty_platform_next_request(anytty_handle_t engine_handle, uint32_t timeout_millis, anytty_buffer_v1 *out_request_proto);
anytty_status_v1 anytty_platform_complete(anytty_handle_t engine_handle, const uint8_t *response_proto, size_t response_length);
anytty_status_v1 anytty_engine_cancel(anytty_handle_t engine_handle, anytty_handle_t operation_handle);
anytty_status_v1 anytty_engine_close_session(anytty_handle_t engine_handle, anytty_handle_t session_handle);
anytty_status_v1 anytty_engine_release(anytty_handle_t engine_handle, anytty_handle_t handle);
anytty_status_v1 anytty_engine_close(anytty_handle_t engine_handle);
anytty_status_v1 anytty_buffer_free(anytty_handle_t buffer_handle);

#ifdef __cplusplus
}
#endif

#endif
