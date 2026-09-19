/**
 * TUI v2 SDK for Node.js/TypeScript.
 *
 * The runtime is plain modern JavaScript (no build step); these declarations
 * give TypeScript consumers the full typed surface. `ChromeApp`-style
 * widgets are intentionally Python/Go-only for now: the TS layer is core +
 * builder, which is all the conformance suite requires.
 */

export type FrameType = 1 | 2 | 3 | 4 | 5;

export interface Hello {
  schema: number;
  view_id: string;
  epoch: number;
  cols: number;
  rows: number;
  components: string[];
  events: string[];
  methods: string[];
  features: Record<string, boolean>;
  limits: Record<string, number>;
}

export interface KeyEvent { kind: 'key'; id: string; key: string; char: string }
export interface PasteEvent { kind: 'paste'; id: string; text: string }
export interface MouseEvent { kind: 'mouse'; action: string; button: string; x: number; y: number; node: string }
export interface WheelEvent { kind: 'wheel'; delta: number; x: number; y: number; node: string }
export interface ResizeEvent { kind: 'resize'; cols: number; rows: number }
export interface Source {
  id: string; kind: string; title: string; endpoint: string; terminal_id: string;
  attached: boolean; exited: boolean; exit_code: number; health: string;
  resize_owner: string; owner_epoch: number; last_seen_ms: number;
}
export interface SourcesEvent { kind: 'sources'; items: Source[] }
export interface NoticeEvent { kind: 'notice'; level: string; message: string }
export interface ComponentEvent { kind: 'component'; source: string; name: string; value: string }
export interface ViewRejectedEvent { kind: 'view_rejected'; epoch: number; rev: number; reason: string }
export type TuiEvent = KeyEvent | PasteEvent | MouseEvent | WheelEvent | ResizeEvent |
  SourcesEvent | NoticeEvent | ComponentEvent | ViewRejectedEvent;

export interface MethodData { rows: string[]; text: string; endpoint: string; id: string }
export interface Response {
  request_id: number;
  epoch: number;
  ok: boolean;
  data: Partial<MethodData>;
  error: string;
}

/** A view-tree node: protocol keys, all optional (PROTOCOL §2). */
export interface Box {
  id?: string;
  size?: [number, number, number];
  pos?: [number, number];
  flow?: 'row' | 'col' | 'stack';
  visible?: boolean;
  cursor?: { row: number; col: number; shape?: string; visible?: boolean };
  content?: { text?: string; lines?: string[]; self?: string; props?: Record<string, string> };
  input?: string[];
  focused?: boolean;
  children?: Box[];
  style?: string;
}

export declare class App {
  client: Client | null;
  emit(method: string, params?: Record<string, unknown>, onResponse?: (response: Response) => void): number;
  log(level: string, message: string): void;
  onHello(hello: Hello): void;
  onEvent(event: TuiEvent): void;
  onKey(event: KeyEvent): void;
  onPaste(event: PasteEvent): void;
  onMouse(event: MouseEvent): void;
  onWheel(event: WheelEvent): void;
  onResize(event: ResizeEvent): void;
  onSources(event: SourcesEvent): void;
  onNotice(event: NoticeEvent): void;
  onComponent(event: ComponentEvent): void;
  onViewRejected(event: ViewRejectedEvent): void;
  onResponse(response: Response): void;
}

export declare class Client {
  constructor(streamIn?: NodeJS.ReadableStream, streamOut?: NodeJS.WritableStream,
              log?: (level: string, message: string) => void);
  hello: Hello | null;
  epoch: number;
  rev: number;
  requestId: number;
  run(app: App): Promise<number>;
  emit(method: string, params?: Record<string, unknown>, onResponse?: (response: Response) => void): number;
  commit(root: Box, claim?: string[], allKeys?: boolean): number;
  log(level: string, message: string): void;
}

export declare class Node {
  constructor(flow?: string);
  id(value: string): this;
  size(width?: number, height?: number, flex?: number): this;
  width(value: number): this;
  height(value: number): this;
  flex(value: number): this;
  pos(x: number, y: number): this;
  flow(value: string): this;
  style(value: string): this;
  focused(value?: boolean): this;
  input(...kinds: string[]): this;
  cursor(row?: number, col?: number, shape?: string, visible?: boolean | null): this;
  content(value: string): this;
  lines(...values: string[]): this;
  selfRef(sourceId: string): this;
  props(values: Record<string, string>): this;
  visible(value: boolean): this;
  child(...nodes: Node[]): this;
  build(): Box;
}

export declare function box(flow?: string): Node;
export declare function col(...children: Node[]): Node;
export declare function row(...children: Node[]): Node;
export declare function stack(...children: Node[]): Node;
export declare function text(value: string): Node;
export declare function terminal(sourceId: string): Node;
export declare function divider(vertical?: boolean, length?: number): Node;

export declare function displayWidth(value: string): number;
export declare function truncate(value: string, maxWidth: number): string;
export declare function centerPad(value: string, cells: number): string;
export declare function clamp(value: number, low: number, high: number): number;

export declare const HELLO: 1;
export declare const VIEW: 2;
export declare const EVENT: 3;
export declare const RESULT: 4;
export declare const RESPONSE: 5;
export declare const MAX_MESSAGE_BYTES: number;
export declare class WireError extends Error {}
export declare function decodeHello(data: Uint8Array): Hello;
export declare function decodeEvent(data: Uint8Array): TuiEvent;
export declare function decodeResponse(data: Uint8Array): Response;
export declare function encodeView(epoch: number, rev: number, claim: string[], allKeys: boolean, root: Box): Buffer;
export declare function encodeResult(requestId: number, epoch: number, method: string, params?: Record<string, unknown>): Buffer;
export declare function frame(frameType: FrameType, payload: Uint8Array): Buffer;
export declare class FrameReader {
  constructor(stream: NodeJS.ReadableStream);
  next(): Promise<{ type: FrameType; payload: Buffer } | null>;
}
