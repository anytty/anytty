#ifndef ANYTTY_TERMINAL_INPUT_H
#define ANYTTY_TERMINAL_INPUT_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define ANYTTY_TERMINAL_INPUT_ABI_VERSION 1u

typedef int32_t anytty_terminal_status_v1;
#define ANYTTY_TERMINAL_STATUS_OK 0
#define ANYTTY_TERMINAL_STATUS_INVALID_ARGUMENT 1
#define ANYTTY_TERMINAL_STATUS_OUT_OF_MEMORY 2
#define ANYTTY_TERMINAL_STATUS_OUT_OF_SPACE 3
#define ANYTTY_TERMINAL_STATUS_REJECTED 4
#define ANYTTY_TERMINAL_STATUS_INTERNAL 5

typedef uint32_t anytty_terminal_key_action_v1;
#define ANYTTY_TERMINAL_KEY_RELEASE 0u
#define ANYTTY_TERMINAL_KEY_PRESS 1u
#define ANYTTY_TERMINAL_KEY_REPEAT 2u

typedef uint16_t anytty_terminal_modifiers_v1;
#define ANYTTY_TERMINAL_MOD_SHIFT (1u << 0)
#define ANYTTY_TERMINAL_MOD_CTRL (1u << 1)
#define ANYTTY_TERMINAL_MOD_ALT (1u << 2)
#define ANYTTY_TERMINAL_MOD_SUPER (1u << 3)
#define ANYTTY_TERMINAL_MOD_CAPS_LOCK (1u << 4)
#define ANYTTY_TERMINAL_MOD_NUM_LOCK (1u << 5)
#define ANYTTY_TERMINAL_MOD_SHIFT_RIGHT (1u << 6)
#define ANYTTY_TERMINAL_MOD_CTRL_RIGHT (1u << 7)
#define ANYTTY_TERMINAL_MOD_ALT_RIGHT (1u << 8)
#define ANYTTY_TERMINAL_MOD_SUPER_RIGHT (1u << 9)

typedef uint32_t anytty_terminal_mouse_action_v1;
#define ANYTTY_TERMINAL_MOUSE_PRESS 0u
#define ANYTTY_TERMINAL_MOUSE_RELEASE 1u
#define ANYTTY_TERMINAL_MOUSE_MOTION 2u

typedef uint32_t anytty_terminal_mouse_button_v1;
#define ANYTTY_TERMINAL_MOUSE_BUTTON_NONE 0u
#define ANYTTY_TERMINAL_MOUSE_BUTTON_LEFT 1u
#define ANYTTY_TERMINAL_MOUSE_BUTTON_RIGHT 2u
#define ANYTTY_TERMINAL_MOUSE_BUTTON_MIDDLE 3u
#define ANYTTY_TERMINAL_MOUSE_BUTTON_SCROLL_UP 4u
#define ANYTTY_TERMINAL_MOUSE_BUTTON_SCROLL_DOWN 5u

typedef struct anytty_terminal_input_v1 anytty_terminal_input_v1;

typedef struct anytty_terminal_buffer_v1 {
  uint8_t *data;
  size_t length;
} anytty_terminal_buffer_v1;

/* This is a sized struct. Zero unknown trailing fields before calling. */
typedef struct anytty_terminal_modes_v1 {
  size_t size;
  bool application_cursor;
  bool application_keypad;
  bool alt_esc_prefix;
  bool modify_other_keys_state_2;
  uint16_t kitty_keyboard_flags;
  bool backarrow_key;
  bool mouse_x10;
  bool mouse_normal;
  bool mouse_button_event;
  bool mouse_any_event;
  bool mouse_sgr;
  bool bracketed_paste;
} anytty_terminal_modes_v1;

/* Pixel geometry used by the Ghostty mouse encoder. */
typedef struct anytty_terminal_geometry_v1 {
  size_t size;
  uint32_t screen_width;
  uint32_t screen_height;
  uint32_t cell_width;
  uint32_t cell_height;
  uint32_t padding_top;
  uint32_t padding_bottom;
  uint32_t padding_right;
  uint32_t padding_left;
} anytty_terminal_geometry_v1;

uint32_t anytty_terminal_input_abi_version(void);
anytty_terminal_status_v1 anytty_terminal_input_new(
    anytty_terminal_input_v1 **out_input);
void anytty_terminal_input_free(anytty_terminal_input_v1 *input);
anytty_terminal_status_v1 anytty_terminal_input_set_modes(
    anytty_terminal_input_v1 *input,
    const anytty_terminal_modes_v1 *modes);
anytty_terminal_status_v1 anytty_terminal_input_set_geometry(
    anytty_terminal_input_v1 *input,
    const anytty_terminal_geometry_v1 *geometry);

/* hid_usage is a USB HID usage including its page, as exposed by Flutter's
 * PhysicalKeyboardKey.usbHidUsage. utf8 is borrowed for this call only. */
anytty_terminal_status_v1 anytty_terminal_input_encode_key(
    anytty_terminal_input_v1 *input,
    uint64_t hid_usage,
    anytty_terminal_key_action_v1 action,
    anytty_terminal_modifiers_v1 modifiers,
    uint32_t unshifted_codepoint,
    bool composing,
    const uint8_t *utf8,
    size_t utf8_length,
    anytty_terminal_buffer_v1 *out_buffer);

anytty_terminal_status_v1 anytty_terminal_input_encode_mouse(
    anytty_terminal_input_v1 *input,
    anytty_terminal_mouse_action_v1 action,
    anytty_terminal_mouse_button_v1 button,
    anytty_terminal_modifiers_v1 modifiers,
    float x,
    float y,
    anytty_terminal_buffer_v1 *out_buffer);

/* Unsafe multiline paste returns REJECTED until allow_unsafe is true. */
anytty_terminal_status_v1 anytty_terminal_input_encode_paste(
    anytty_terminal_input_v1 *input,
    const uint8_t *utf8,
    size_t utf8_length,
    bool allow_unsafe,
    anytty_terminal_buffer_v1 *out_buffer);

void anytty_terminal_buffer_free(anytty_terminal_buffer_v1 buffer);

#ifdef __cplusplus
}
#endif

#endif
