#include "anytty_terminal_input.h"

#include <assert.h>
#include <string.h>

#define HID(value) (0x00070000u | (value))

static void assert_key(anytty_terminal_input_v1 *input, uint64_t hid,
                       const char *expected, size_t expected_length) {
  anytty_terminal_buffer_v1 output = {0};
  assert(anytty_terminal_input_encode_key(
             input, hid, ANYTTY_TERMINAL_KEY_PRESS, 0, 0, false,
             NULL, 0, &output) == ANYTTY_TERMINAL_STATUS_OK);
  assert(output.length == expected_length);
  assert(memcmp(output.data, expected, expected_length) == 0);
  anytty_terminal_buffer_free(output);
}

int main(void) {
  assert(anytty_terminal_input_abi_version() == 1);
  anytty_terminal_input_v1 *input = NULL;
  assert(anytty_terminal_input_new(&input) == ANYTTY_TERMINAL_STATUS_OK);

  anytty_terminal_modes_v1 modes = {0};
  modes.size = sizeof(modes);
  modes.application_cursor = true;
  modes.alt_esc_prefix = true;
  modes.bracketed_paste = true;
  assert(anytty_terminal_input_set_modes(input, &modes) ==
         ANYTTY_TERMINAL_STATUS_OK);

  anytty_terminal_buffer_v1 output = {0};
  assert(anytty_terminal_input_encode_key(
             input, HID(0x52), ANYTTY_TERMINAL_KEY_PRESS, 0, 0, false,
             NULL, 0, &output) == ANYTTY_TERMINAL_STATUS_OK);
  assert(output.length == 3);
  assert(memcmp(output.data, "\x1bOA", 3) == 0);
  anytty_terminal_buffer_free(output);

  assert_key(input, HID(0x28), "\r", 1);
  assert_key(input, HID(0x29), "\x1b", 1);
  assert_key(input, HID(0x2a), "\x7f", 1);
  assert_key(input, HID(0x2b), "\t", 1);
  assert_key(input, HID(0x4a), "\x1bOH", 3);
  assert_key(input, HID(0x4b), "\x1b[5~", 4);
  assert_key(input, HID(0x4c), "\x1b[3~", 4);
  assert_key(input, HID(0x4d), "\x1bOF", 3);
  assert_key(input, HID(0x4e), "\x1b[6~", 4);

  modes.mouse_normal = true;
  modes.mouse_sgr = true;
  assert(anytty_terminal_input_set_modes(input, &modes) ==
         ANYTTY_TERMINAL_STATUS_OK);
  anytty_terminal_geometry_v1 geometry = {0};
  geometry.size = sizeof(geometry);
  geometry.screen_width = 8440;
  geometry.screen_height = 4000;
  geometry.cell_width = 844;
  geometry.cell_height = 2000;
  assert(anytty_terminal_input_set_geometry(input, &geometry) ==
         ANYTTY_TERMINAL_STATUS_OK);

  output = (anytty_terminal_buffer_v1){0};
  assert(anytty_terminal_input_encode_mouse(
             input, ANYTTY_TERMINAL_MOUSE_PRESS,
             ANYTTY_TERMINAL_MOUSE_BUTTON_LEFT, 0, 1266, 3000, &output) ==
         ANYTTY_TERMINAL_STATUS_OK);
  assert(output.length > 6);
  assert(memcmp(output.data, "\x1b[<0;", 5) == 0);
  assert(output.data[output.length - 1] == 'M');
  anytty_terminal_buffer_free(output);

  output = (anytty_terminal_buffer_v1){0};
  assert(anytty_terminal_input_encode_mouse(
             input, ANYTTY_TERMINAL_MOUSE_RELEASE,
             ANYTTY_TERMINAL_MOUSE_BUTTON_LEFT, 0, 1266, 3000, &output) ==
         ANYTTY_TERMINAL_STATUS_OK);
  assert(output.length > 6);
  assert(memcmp(output.data, "\x1b[<0;", 5) == 0);
  assert(output.data[output.length - 1] == 'm');
  anytty_terminal_buffer_free(output);

  const uint8_t paste[] = "hello";
  output = (anytty_terminal_buffer_v1){0};
  assert(anytty_terminal_input_encode_paste(
             input, paste, sizeof(paste) - 1, false, &output) ==
         ANYTTY_TERMINAL_STATUS_OK);
  assert(output.length == 17);
  assert(memcmp(output.data, "\x1b[200~hello\x1b[201~", 17) == 0);
  anytty_terminal_buffer_free(output);

  const uint8_t unsafe[] = "echo yes\n";
  assert(anytty_terminal_input_encode_paste(
             input, unsafe, sizeof(unsafe) - 1, false, &output) ==
         ANYTTY_TERMINAL_STATUS_REJECTED);

  anytty_terminal_input_free(input);
  return 0;
}
