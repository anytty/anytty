// anytty-managed-agent-hook-v1
// Official OpenCode event plugin; never changes permissions or tool arguments.
import { spawn } from "node:child_process";

const executable = "__ANYTTY_EXECUTABLE__";
export const AnyTTYAgents = async ({ directory }) => {
  if (!process.env.ANYTTY_TERMINAL_ID || !process.env.ANYTTY_DAEMON_SOCKET) return {};
  const epoch = Date.now();
  let sequence = 0;
  const sessions = new Map();
  const pending = new Map();
  let draining = false;
  let retryTimer;
  const send = (report) => new Promise((resolve) => {
    const child = spawn(executable, ["plugin", "agents", "hook", "opencode"], {
      stdio: ["pipe", "ignore", "ignore"], timeout: 1500, killSignal: "SIGKILL",
    });
    child.on("error", () => resolve(false));
    child.on("close", (code) => resolve(code === 0));
    child.stdin.on("error", () => {});
    child.stdin.end(JSON.stringify(report));
  });
  const drain = async () => {
    if (draining) return;
    clearTimeout(retryTimer);
    draining = true;
    try {
      while (pending.size) {
        const now = Date.now();
        const ready = [...pending].find(([, item]) => item.readyAt <= now);
        if (!ready) {
          const delay = Math.max(1, Math.min(...[...pending.values()].map(item => item.readyAt)) - now);
          retryTimer = setTimeout(() => { void drain(); }, delay);
          retryTimer.unref?.();
          return;
        }
        const [id, item] = ready;
        pending.delete(id);
        const delivered = await send(item.report).catch(() => false);
        // At most three attempts, always asynchronous. Never overwrite a more
        // recent complete snapshot queued while this attempt was in flight.
        if (!delivered && item.attempt < 3 && !pending.has(id)) {
          pending.set(id, { report: item.report, attempt: item.attempt + 1,
            readyAt: Date.now() + 100 * item.attempt });
        }
      }
    } finally { draining = false; }
  };
  return {
    event: async ({ event }) => {
      const p = event.properties ?? {};
      const sessionID = p.sessionID ?? p.info?.id;
      if (!sessionID) return;
      let kind = "status", status = "", permissionID;
      switch (event.type) {
        case "session.created": kind = "start"; status = "idle"; break;
        case "session.updated": kind = "metadata"; break;
        case "session.status":
          status = ({ busy: "working", retry: "working", idle: "idle" })[p.status?.type] ?? "unknown";
          break;
        case "session.idle": status = "idle"; break;
        case "session.error": status = "error"; break;
        case "session.deleted": status = "exited"; break;
        case "permission.asked": kind = event.type; permissionID = p.id; break;
        case "permission.replied": kind = event.type; permissionID = p.requestID; break;
        default: return;
      }
      if (kind.startsWith("permission.") && !permissionID) return;
      const state = sessions.get(sessionID) ?? { status: "unknown", permissions: new Set(), title: "", project: directory };
      if (status) state.status = status;
      if (kind === "permission.asked") state.permissions.add(permissionID);
      if (kind === "permission.replied") state.permissions.delete(permissionID);
      if (status === "exited") state.permissions.clear();
      if (p.info?.title) state.title = p.info.title;
      if (p.info?.directory) state.project = p.info.directory;
      sessions.set(sessionID, state);
      const report = {
        agent: "opencode", session_id: sessionID, project: state.project,
        title: state.title, kind, status: state.status, permission_id: permissionID,
        epoch, sequence: ++sequence, full_state: true, pending_permissions: [...state.permissions],
      };
      // One complete pending snapshot per session, not an unbounded event queue.
      // Coalescing cannot lose a permission reply or regress the latest status.
      pending.set(sessionID, { report, attempt: 1, readyAt: Date.now() });
      void drain();
    },
  };
};
