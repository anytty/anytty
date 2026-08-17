import { useEffect, useState } from "react";
import { Check, Copy } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

type InstallTarget = "unix" | "windows";

interface InstallTabsProps {
  ariaLabel: string;
  unixLabel: string;
  windowsLabel: string;
  unixCommand: string;
  windowsCommand: string;
  copyLabel: string;
  copiedLabel: string;
}

export function InstallTabs({
  ariaLabel,
  unixLabel,
  windowsLabel,
  unixCommand,
  windowsCommand,
  copyLabel,
  copiedLabel,
}: InstallTabsProps) {
  const [target, setTarget] = useState<InstallTarget>("unix");

  useEffect(() => {
    if (/Windows/i.test(navigator.userAgent)) setTarget("windows");
  }, []);

  return (
    <Tabs value={target} onValueChange={(value) => setTarget(value as InstallTarget)} className="min-w-0 w-full">
      <TabsList aria-label={ariaLabel} className="grid h-11 w-full grid-cols-2 bg-muted/80 sm:w-fit">
        <TabsTrigger value="unix" className="h-9 px-4 text-xs font-normal sm:text-sm">
          {unixLabel}
        </TabsTrigger>
        <TabsTrigger value="windows" className="h-9 px-4 text-xs font-normal sm:text-sm">
          {windowsLabel}
        </TabsTrigger>
      </TabsList>
      <TabsContent value="unix" className="mt-3 min-w-0">
        <Command command={unixCommand} prompt="$" copyLabel={copyLabel} copiedLabel={copiedLabel} />
      </TabsContent>
      <TabsContent value="windows" className="mt-3 min-w-0">
        <Command command={windowsCommand} prompt="PS>" copyLabel={copyLabel} copiedLabel={copiedLabel} />
      </TabsContent>
    </Tabs>
  );
}

function Command({ command, prompt, copyLabel, copiedLabel }: {
  command: string;
  prompt: string;
  copyLabel: string;
  copiedLabel: string;
}) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(command);
      } else {
        const field = document.createElement("textarea");
        field.value = command;
        field.setAttribute("readonly", "");
        field.className = "pointer-events-none fixed left-0 top-0 size-px opacity-0";
        document.body.appendChild(field);
        field.select();
        const copied = document.execCommand("copy");
        field.remove();
        if (!copied) throw new Error("Copy command was rejected");
      }
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  }

  return (
    <div className="flex min-h-14 min-w-0 max-w-full items-center gap-3 rounded-lg bg-zinc-950 px-3 text-zinc-100 shadow-sm ring-1 ring-white/10 sm:px-4">
      <span className="shrink-0 font-mono text-xs text-emerald-400" aria-hidden="true">{prompt}</span>
      <code className="min-w-0 flex-1 overflow-x-auto whitespace-nowrap py-4 font-mono text-xs font-normal sm:text-sm">
        {command}
      </code>
      <TooltipProvider delayDuration={250}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="size-11 shrink-0 text-zinc-300 hover:bg-white/10 hover:text-white"
              aria-label={copied ? copiedLabel : copyLabel}
              onClick={copy}
            >
              {copied ? <Check aria-hidden="true" /> : <Copy aria-hidden="true" />}
            </Button>
          </TooltipTrigger>
          <TooltipContent>{copied ? copiedLabel : copyLabel}</TooltipContent>
        </Tooltip>
      </TooltipProvider>
      <span className="sr-only" aria-live="polite">{copied ? copiedLabel : ""}</span>
    </div>
  );
}
