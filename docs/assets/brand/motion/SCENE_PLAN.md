# Scenario-First Motion Plan

Revision 3 adds full-body poses without the phone. These are proposed character extensions,
not replacements for the approved static logo. The original head, eyes, palette, and tail
remain the identity anchors; open hands and small articulated legs/feet are new drawings.

| Scene | Real context | Action sequence | Playback rule |
| --- | --- | --- | --- |
| Welcome | First session or an intentional greeting | Settle into standing pose, notice the viewer, wave an open hand, lower it | Once, then still; not on every navigation |
| Running | An active task, connection attempt, or transfer with unknown duration | Forward lean, alternating foot contacts and arm counter-swing, head and tail follow | Loop only while work is pending; never imply measurable progress |
| Searching | File search or device discovery | Lower stance, lean and glance left, shift gaze right, return | Loop while searching; stop for results or an explicit empty state |
| Wake | Returning from background after inactivity | Drowsy crouch, open eyes, extend both arms, lift head, settle alert | Once; animation is not evidence that the network recovered |

The phone-based connection, processing, history, success, and failure actions remain available.
Do not use a wave as proof of a successful connection, or a running loop for a terminal failure.
Actual status text, retry/cancel controls, and accessibility announcements still come from App state.

## Visual Direction

Preserve cyan `#1bb3ff`, near-black `#070a0b`, green `#2bce99`, and white eyes.
Keep the preview's system typography and restrained controls. The distinguishing change is the
whole-body silhouette and choreography, not more decorative effects or extra props.
Feet make contact before the torso settles; arms counter the stride; head/tail motion trails
the primary gesture. Waiting loops have matching endpoints. Greetings and wake cues do not loop.

## Review Gates

- Each new pose must read without the phone or screen glyphs.
- Preserve all five existing scene names and triggers.
- No white matte, clipped limbs, detached joints, or blank reduced-motion frames.
- Every new animated property is reset in all other states.
- Test desktop, phone portrait/landscape, light/dark, pause/replay, and reduced motion.
- Do not wire these samples into Flutter until the poses are approved.
