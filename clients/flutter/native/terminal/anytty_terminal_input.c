#include "anytty_terminal_input.h"

#include <stdlib.h>
#include <string.h>

#include <ghostty/vt.h>

#define USB_KEYBOARD_USAGE_PAGE 0x00070000u
#define USB_USAGE(value) (USB_KEYBOARD_USAGE_PAGE | (value))

struct anytty_terminal_input_v1 {
  GhosttyKeyEncoder key_encoder;
  GhosttyKeyEvent key_event;
  GhosttyMouseEncoder mouse_encoder;
  GhosttyMouseEvent mouse_event;
  bool bracketed_paste;
};

static anytty_terminal_status_v1 status_from_ghostty(GhosttyResult result) {
  switch (result) {
    case GHOSTTY_SUCCESS:
      return ANYTTY_TERMINAL_STATUS_OK;
    case GHOSTTY_OUT_OF_MEMORY:
      return ANYTTY_TERMINAL_STATUS_OUT_OF_MEMORY;
    case GHOSTTY_OUT_OF_SPACE:
      return ANYTTY_TERMINAL_STATUS_OUT_OF_SPACE;
    case GHOSTTY_REJECTED:
      return ANYTTY_TERMINAL_STATUS_REJECTED;
    case GHOSTTY_INVALID_VALUE:
      return ANYTTY_TERMINAL_STATUS_INVALID_ARGUMENT;
    default:
      return ANYTTY_TERMINAL_STATUS_INTERNAL;
  }
}

static GhosttyKey key_from_hid(uint64_t usage) {
  if (usage >= USB_USAGE(0x04) && usage <= USB_USAGE(0x1d)) {
    return (GhosttyKey)(GHOSTTY_KEY_A + (usage - USB_USAGE(0x04)));
  }
  if (usage >= USB_USAGE(0x1e) && usage <= USB_USAGE(0x26)) {
    return (GhosttyKey)(GHOSTTY_KEY_DIGIT_1 + (usage - USB_USAGE(0x1e)));
  }
  if (usage == USB_USAGE(0x27)) return GHOSTTY_KEY_DIGIT_0;
  if (usage >= USB_USAGE(0x3a) && usage <= USB_USAGE(0x45)) {
    return (GhosttyKey)(GHOSTTY_KEY_F1 + (usage - USB_USAGE(0x3a)));
  }
  if (usage >= USB_USAGE(0x68) && usage <= USB_USAGE(0x73)) {
    return (GhosttyKey)(GHOSTTY_KEY_F13 + (usage - USB_USAGE(0x68)));
  }
  switch (usage) {
    case USB_USAGE(0x28): return GHOSTTY_KEY_ENTER;
    case USB_USAGE(0x29): return GHOSTTY_KEY_ESCAPE;
    case USB_USAGE(0x2a): return GHOSTTY_KEY_BACKSPACE;
    case USB_USAGE(0x2b): return GHOSTTY_KEY_TAB;
    case USB_USAGE(0x2c): return GHOSTTY_KEY_SPACE;
    case USB_USAGE(0x2d): return GHOSTTY_KEY_MINUS;
    case USB_USAGE(0x2e): return GHOSTTY_KEY_EQUAL;
    case USB_USAGE(0x2f): return GHOSTTY_KEY_BRACKET_LEFT;
    case USB_USAGE(0x30): return GHOSTTY_KEY_BRACKET_RIGHT;
    case USB_USAGE(0x31): return GHOSTTY_KEY_BACKSLASH;
    case USB_USAGE(0x33): return GHOSTTY_KEY_SEMICOLON;
    case USB_USAGE(0x34): return GHOSTTY_KEY_QUOTE;
    case USB_USAGE(0x35): return GHOSTTY_KEY_BACKQUOTE;
    case USB_USAGE(0x36): return GHOSTTY_KEY_COMMA;
    case USB_USAGE(0x37): return GHOSTTY_KEY_PERIOD;
    case USB_USAGE(0x38): return GHOSTTY_KEY_SLASH;
    case USB_USAGE(0x46): return GHOSTTY_KEY_PRINT_SCREEN;
    case USB_USAGE(0x47): return GHOSTTY_KEY_SCROLL_LOCK;
    case USB_USAGE(0x48): return GHOSTTY_KEY_PAUSE;
    case USB_USAGE(0x49): return GHOSTTY_KEY_INSERT;
    case USB_USAGE(0x4a): return GHOSTTY_KEY_HOME;
    case USB_USAGE(0x4b): return GHOSTTY_KEY_PAGE_UP;
    case USB_USAGE(0x4c): return GHOSTTY_KEY_DELETE;
    case USB_USAGE(0x4d): return GHOSTTY_KEY_END;
    case USB_USAGE(0x4e): return GHOSTTY_KEY_PAGE_DOWN;
    case USB_USAGE(0x4f): return GHOSTTY_KEY_ARROW_RIGHT;
    case USB_USAGE(0x50): return GHOSTTY_KEY_ARROW_LEFT;
    case USB_USAGE(0x51): return GHOSTTY_KEY_ARROW_DOWN;
    case USB_USAGE(0x52): return GHOSTTY_KEY_ARROW_UP;
    case USB_USAGE(0x53): return GHOSTTY_KEY_NUM_LOCK;
    case USB_USAGE(0x54): return GHOSTTY_KEY_NUMPAD_DIVIDE;
    case USB_USAGE(0x55): return GHOSTTY_KEY_NUMPAD_MULTIPLY;
    case USB_USAGE(0x56): return GHOSTTY_KEY_NUMPAD_SUBTRACT;
    case USB_USAGE(0x57): return GHOSTTY_KEY_NUMPAD_ADD;
    case USB_USAGE(0x58): return GHOSTTY_KEY_NUMPAD_ENTER;
    case USB_USAGE(0x59): return GHOSTTY_KEY_NUMPAD_1;
    case USB_USAGE(0x5a): return GHOSTTY_KEY_NUMPAD_2;
    case USB_USAGE(0x5b): return GHOSTTY_KEY_NUMPAD_3;
    case USB_USAGE(0x5c): return GHOSTTY_KEY_NUMPAD_4;
    case USB_USAGE(0x5d): return GHOSTTY_KEY_NUMPAD_5;
    case USB_USAGE(0x5e): return GHOSTTY_KEY_NUMPAD_6;
    case USB_USAGE(0x5f): return GHOSTTY_KEY_NUMPAD_7;
    case USB_USAGE(0x60): return GHOSTTY_KEY_NUMPAD_8;
    case USB_USAGE(0x61): return GHOSTTY_KEY_NUMPAD_9;
    case USB_USAGE(0x62): return GHOSTTY_KEY_NUMPAD_0;
    case USB_USAGE(0x63): return GHOSTTY_KEY_NUMPAD_DECIMAL;
    case USB_USAGE(0x64): return GHOSTTY_KEY_INTL_BACKSLASH;
    case USB_USAGE(0x67): return GHOSTTY_KEY_NUMPAD_EQUAL;
    case USB_USAGE(0xe0): return GHOSTTY_KEY_CONTROL_LEFT;
    case USB_USAGE(0xe1): return GHOSTTY_KEY_SHIFT_LEFT;
    case USB_USAGE(0xe2): return GHOSTTY_KEY_ALT_LEFT;
    case USB_USAGE(0xe3): return GHOSTTY_KEY_META_LEFT;
    case USB_USAGE(0xe4): return GHOSTTY_KEY_CONTROL_RIGHT;
    case USB_USAGE(0xe5): return GHOSTTY_KEY_SHIFT_RIGHT;
    case USB_USAGE(0xe6): return GHOSTTY_KEY_ALT_RIGHT;
    case USB_USAGE(0xe7): return GHOSTTY_KEY_META_RIGHT;
    default: return GHOSTTY_KEY_UNIDENTIFIED;
  }
}

static anytty_terminal_status_v1 copy_encoded(
    GhosttyResult first_result,
    size_t required,
    GhosttyResult (*encode)(void *, char *, size_t, size_t *),
    void *context,
    anytty_terminal_buffer_v1 *out) {
  if (first_result == GHOSTTY_SUCCESS && required == 0) return ANYTTY_TERMINAL_STATUS_OK;
  if (first_result != GHOSTTY_OUT_OF_SPACE) return status_from_ghostty(first_result);
  uint8_t *data = (uint8_t *)malloc(required);
  if (data == NULL) return ANYTTY_TERMINAL_STATUS_OUT_OF_MEMORY;
  size_t written = 0;
  GhosttyResult result = encode(context, (char *)data, required, &written);
  if (result != GHOSTTY_SUCCESS) {
    free(data);
    return status_from_ghostty(result);
  }
  out->data = data;
  out->length = written;
  return ANYTTY_TERMINAL_STATUS_OK;
}

typedef struct {
  GhosttyKeyEncoder encoder;
  GhosttyKeyEvent event;
} key_encode_context;

static GhosttyResult encode_key_again(
    void *opaque, char *data, size_t length, size_t *written) {
  key_encode_context *context = (key_encode_context *)opaque;
  return ghostty_key_encoder_encode(
      context->encoder, context->event, data, length, written);
}

typedef struct {
  GhosttyMouseEncoder encoder;
  GhosttyMouseEvent event;
} mouse_encode_context;

static GhosttyResult encode_mouse_again(
    void *opaque, char *data, size_t length, size_t *written) {
  mouse_encode_context *context = (mouse_encode_context *)opaque;
  return ghostty_mouse_encoder_encode(
      context->encoder, context->event, data, length, written);
}

uint32_t anytty_terminal_input_abi_version(void) {
  return ANYTTY_TERMINAL_INPUT_ABI_VERSION;
}

anytty_terminal_status_v1 anytty_terminal_input_new(
    anytty_terminal_input_v1 **out_input) {
  if (out_input == NULL) return ANYTTY_TERMINAL_STATUS_INVALID_ARGUMENT;
  *out_input = NULL;
  anytty_terminal_input_v1 *input = calloc(1, sizeof(*input));
  if (input == NULL) return ANYTTY_TERMINAL_STATUS_OUT_OF_MEMORY;

  GhosttyResult result = ghostty_key_encoder_new(NULL, &input->key_encoder);
  if (result == GHOSTTY_SUCCESS) {
    result = ghostty_key_event_new(NULL, &input->key_event);
  }
  if (result == GHOSTTY_SUCCESS) {
    result = ghostty_mouse_encoder_new(NULL, &input->mouse_encoder);
  }
  if (result == GHOSTTY_SUCCESS) {
    result = ghostty_mouse_event_new(NULL, &input->mouse_event);
  }
  if (result != GHOSTTY_SUCCESS) {
    anytty_terminal_input_free(input);
    return status_from_ghostty(result);
  }

  bool enabled = true;
  ghostty_key_encoder_setopt(
      input->key_encoder, GHOSTTY_KEY_ENCODER_OPT_ALT_ESC_PREFIX, &enabled);
  GhosttyOptionAsAlt option_as_alt = GHOSTTY_OPTION_AS_ALT_TRUE;
  ghostty_key_encoder_setopt(
      input->key_encoder,
      GHOSTTY_KEY_ENCODER_OPT_MACOS_OPTION_AS_ALT,
      &option_as_alt);
  *out_input = input;
  return ANYTTY_TERMINAL_STATUS_OK;
}

void anytty_terminal_input_free(anytty_terminal_input_v1 *input) {
  if (input == NULL) return;
  ghostty_mouse_event_free(input->mouse_event);
  ghostty_mouse_encoder_free(input->mouse_encoder);
  ghostty_key_event_free(input->key_event);
  ghostty_key_encoder_free(input->key_encoder);
  free(input);
}

anytty_terminal_status_v1 anytty_terminal_input_set_modes(
    anytty_terminal_input_v1 *input,
    const anytty_terminal_modes_v1 *modes) {
  if (input == NULL || modes == NULL || modes->size < sizeof(*modes)) {
    return ANYTTY_TERMINAL_STATUS_INVALID_ARGUMENT;
  }
  ghostty_key_encoder_setopt(input->key_encoder,
      GHOSTTY_KEY_ENCODER_OPT_CURSOR_KEY_APPLICATION,
      &modes->application_cursor);
  ghostty_key_encoder_setopt(input->key_encoder,
      GHOSTTY_KEY_ENCODER_OPT_KEYPAD_KEY_APPLICATION,
      &modes->application_keypad);
  ghostty_key_encoder_setopt(input->key_encoder,
      GHOSTTY_KEY_ENCODER_OPT_ALT_ESC_PREFIX,
      &modes->alt_esc_prefix);
  ghostty_key_encoder_setopt(input->key_encoder,
      GHOSTTY_KEY_ENCODER_OPT_MODIFY_OTHER_KEYS_STATE_2,
      &modes->modify_other_keys_state_2);
  GhosttyKittyKeyFlags kitty_flags = modes->kitty_keyboard_flags;
  ghostty_key_encoder_setopt(input->key_encoder,
      GHOSTTY_KEY_ENCODER_OPT_KITTY_FLAGS,
      &kitty_flags);
  ghostty_key_encoder_setopt(input->key_encoder,
      GHOSTTY_KEY_ENCODER_OPT_BACKARROW_KEY_MODE,
      &modes->backarrow_key);

  GhosttyMouseTrackingMode tracking = GHOSTTY_MOUSE_TRACKING_NONE;
  if (modes->mouse_any_event) tracking = GHOSTTY_MOUSE_TRACKING_ANY;
  else if (modes->mouse_button_event) tracking = GHOSTTY_MOUSE_TRACKING_BUTTON;
  else if (modes->mouse_normal) tracking = GHOSTTY_MOUSE_TRACKING_NORMAL;
  else if (modes->mouse_x10) tracking = GHOSTTY_MOUSE_TRACKING_X10;
  ghostty_mouse_encoder_setopt(
      input->mouse_encoder, GHOSTTY_MOUSE_ENCODER_OPT_EVENT, &tracking);
  GhosttyMouseFormat format = modes->mouse_sgr
      ? GHOSTTY_MOUSE_FORMAT_SGR
      : GHOSTTY_MOUSE_FORMAT_X10;
  ghostty_mouse_encoder_setopt(
      input->mouse_encoder, GHOSTTY_MOUSE_ENCODER_OPT_FORMAT, &format);
  input->bracketed_paste = modes->bracketed_paste;
  return ANYTTY_TERMINAL_STATUS_OK;
}

anytty_terminal_status_v1 anytty_terminal_input_set_geometry(
    anytty_terminal_input_v1 *input,
    const anytty_terminal_geometry_v1 *geometry) {
  if (input == NULL || geometry == NULL ||
      geometry->size < sizeof(*geometry) ||
      geometry->cell_width == 0 || geometry->cell_height == 0) {
    return ANYTTY_TERMINAL_STATUS_INVALID_ARGUMENT;
  }
  GhosttyMouseEncoderSize size = GHOSTTY_INIT_SIZED(GhosttyMouseEncoderSize);
  size.screen_width = geometry->screen_width;
  size.screen_height = geometry->screen_height;
  size.cell_width = geometry->cell_width;
  size.cell_height = geometry->cell_height;
  size.padding_top = geometry->padding_top;
  size.padding_bottom = geometry->padding_bottom;
  size.padding_right = geometry->padding_right;
  size.padding_left = geometry->padding_left;
  ghostty_mouse_encoder_setopt(
      input->mouse_encoder, GHOSTTY_MOUSE_ENCODER_OPT_SIZE, &size);
  return ANYTTY_TERMINAL_STATUS_OK;
}

anytty_terminal_status_v1 anytty_terminal_input_encode_key(
    anytty_terminal_input_v1 *input,
    uint64_t hid_usage,
    anytty_terminal_key_action_v1 action,
    anytty_terminal_modifiers_v1 modifiers,
    uint32_t unshifted_codepoint,
    bool composing,
    const uint8_t *utf8,
    size_t utf8_length,
    anytty_terminal_buffer_v1 *out_buffer) {
  if (input == NULL || out_buffer == NULL || action > ANYTTY_TERMINAL_KEY_REPEAT ||
      (utf8 == NULL && utf8_length != 0)) {
    return ANYTTY_TERMINAL_STATUS_INVALID_ARGUMENT;
  }
  out_buffer->data = NULL;
  out_buffer->length = 0;
  ghostty_key_event_set_key(input->key_event, key_from_hid(hid_usage));
  ghostty_key_event_set_action(input->key_event, (GhosttyKeyAction)action);
  ghostty_key_event_set_mods(input->key_event, (GhosttyMods)modifiers);
  ghostty_key_event_set_consumed_mods(input->key_event, 0);
  ghostty_key_event_set_composing(input->key_event, composing);
  ghostty_key_event_set_unshifted_codepoint(
      input->key_event, unshifted_codepoint);
  ghostty_key_event_set_utf8(
      input->key_event, (const char *)utf8, utf8_length);

  size_t required = 0;
  GhosttyResult result = ghostty_key_encoder_encode(
      input->key_encoder, input->key_event, NULL, 0, &required);
  key_encode_context context = {input->key_encoder, input->key_event};
  return copy_encoded(
      result, required, encode_key_again, &context, out_buffer);
}

anytty_terminal_status_v1 anytty_terminal_input_encode_mouse(
    anytty_terminal_input_v1 *input,
    anytty_terminal_mouse_action_v1 action,
    anytty_terminal_mouse_button_v1 button,
    anytty_terminal_modifiers_v1 modifiers,
    float x,
    float y,
    anytty_terminal_buffer_v1 *out_buffer) {
  if (input == NULL || out_buffer == NULL ||
      action > ANYTTY_TERMINAL_MOUSE_MOTION ||
      button > ANYTTY_TERMINAL_MOUSE_BUTTON_SCROLL_DOWN) {
    return ANYTTY_TERMINAL_STATUS_INVALID_ARGUMENT;
  }
  out_buffer->data = NULL;
  out_buffer->length = 0;
  ghostty_mouse_event_set_action(input->mouse_event, (GhosttyMouseAction)action);
  if (button == ANYTTY_TERMINAL_MOUSE_BUTTON_NONE) {
    ghostty_mouse_event_clear_button(input->mouse_event);
  } else {
    ghostty_mouse_event_set_button(input->mouse_event, (GhosttyMouseButton)button);
  }
  ghostty_mouse_event_set_mods(input->mouse_event, (GhosttyMods)modifiers);
  GhosttyMousePosition position = {.x = x, .y = y};
  ghostty_mouse_event_set_position(input->mouse_event, position);

  size_t required = 0;
  GhosttyResult result = ghostty_mouse_encoder_encode(
      input->mouse_encoder, input->mouse_event, NULL, 0, &required);
  mouse_encode_context context = {input->mouse_encoder, input->mouse_event};
  return copy_encoded(
      result, required, encode_mouse_again, &context, out_buffer);
}

anytty_terminal_status_v1 anytty_terminal_input_encode_paste(
    anytty_terminal_input_v1 *input,
    const uint8_t *utf8,
    size_t utf8_length,
    bool allow_unsafe,
    anytty_terminal_buffer_v1 *out_buffer) {
  if (input == NULL || out_buffer == NULL ||
      (utf8 == NULL && utf8_length != 0)) {
    return ANYTTY_TERMINAL_STATUS_INVALID_ARGUMENT;
  }
  out_buffer->data = NULL;
  out_buffer->length = 0;
  if (!allow_unsafe &&
      !ghostty_paste_is_safe((const char *)utf8, utf8_length)) {
    return ANYTTY_TERMINAL_STATUS_REJECTED;
  }

  char *input_copy = NULL;
  if (utf8_length != 0) {
    input_copy = (char *)malloc(utf8_length);
    if (input_copy == NULL) return ANYTTY_TERMINAL_STATUS_OUT_OF_MEMORY;
    memcpy(input_copy, utf8, utf8_length);
  }
  size_t required = 0;
  GhosttyResult result = ghostty_paste_encode(
      input_copy, utf8_length, input->bracketed_paste,
      NULL, 0, &required);
  free(input_copy);
  if (result != GHOSTTY_OUT_OF_SPACE && result != GHOSTTY_SUCCESS) {
    return status_from_ghostty(result);
  }
  if (required == 0) return ANYTTY_TERMINAL_STATUS_OK;

  input_copy = (char *)malloc(utf8_length);
  uint8_t *encoded = (uint8_t *)malloc(required);
  if ((utf8_length != 0 && input_copy == NULL) || encoded == NULL) {
    free(input_copy);
    free(encoded);
    return ANYTTY_TERMINAL_STATUS_OUT_OF_MEMORY;
  }
  if (utf8_length != 0) memcpy(input_copy, utf8, utf8_length);
  size_t written = 0;
  result = ghostty_paste_encode(
      input_copy, utf8_length, input->bracketed_paste,
      (char *)encoded, required, &written);
  free(input_copy);
  if (result != GHOSTTY_SUCCESS) {
    free(encoded);
    return status_from_ghostty(result);
  }
  out_buffer->data = encoded;
  out_buffer->length = written;
  return ANYTTY_TERMINAL_STATUS_OK;
}

void anytty_terminal_buffer_free(anytty_terminal_buffer_v1 buffer) {
  free(buffer.data);
}
