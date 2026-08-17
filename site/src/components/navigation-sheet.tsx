import { ArrowDownToLine, BookOpen, Cloud, Menu, PanelLeft } from "lucide-react";

import { GithubIcon } from "@/components/github-icon";

import { Button, buttonVariants } from "@/components/ui/button";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { cn } from "@/lib/utils";

interface NavigationItem {
  label: string;
  href: string;
  current?: boolean;
  emphasis?: boolean;
  kind?: "default" | "github" | "cloud" | "download";
}

interface NavigationSheetProps {
  label: string;
  title: string;
  description?: string;
  closeLabel: string;
  items: NavigationItem[];
  trigger?: "menu" | "docs";
}

export function NavigationSheet({
  label,
  title,
  description,
  closeLabel,
  items,
  trigger = "menu",
}: NavigationSheetProps) {
  const TriggerIcon = trigger === "docs" ? PanelLeft : Menu;

  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button
          type="button"
          variant={trigger === "docs" ? "outline" : "ghost"}
          size={trigger === "docs" ? "default" : "icon"}
          className={cn("h-11", trigger === "docs" ? "justify-start px-3 font-normal lg:hidden" : "w-11 rounded-full bg-[var(--chip-soft)] text-zinc-200 hover:bg-[var(--chip)] hover:text-zinc-100 lg:hidden")}
          aria-label={label}
        >
          <TriggerIcon aria-hidden="true" />
          {trigger === "docs" && <span>{label}</span>}
        </Button>
      </SheetTrigger>
      <SheetContent side="right" closeLabel={closeLabel} className="w-[min(88vw,22rem)] border-0 bg-zinc-950 text-zinc-100 p-0">
        <div className="flex h-full flex-col px-5 pb-6 pt-16">
          <SheetHeader className="text-left">
            <SheetTitle className="text-base font-medium">{title}</SheetTitle>
            {description && <SheetDescription className="font-normal leading-6">{description}</SheetDescription>}
          </SheetHeader>
          <nav className="mt-8 flex flex-col gap-1" aria-label={label}>
            {items.map((item) => {
              const Icon = item.kind === "github"
                ? GithubIcon
                : item.kind === "cloud"
                  ? Cloud
                  : item.kind === "download"
                    ? ArrowDownToLine
                    : item.kind === "default" && trigger === "docs"
                      ? BookOpen
                      : null;
              return (
                <SheetClose asChild key={`${item.label}-${item.href}`}>
                  <a
                    href={item.href}
                    aria-current={item.current ? "page" : undefined}
                    className={cn(
                      buttonVariants({ variant: item.emphasis ? "default" : item.current ? "secondary" : "ghost" }),
                      "h-11 justify-start rounded-full px-3 text-sm font-normal",
                    )}
                  >
                    {Icon && <Icon aria-hidden="true" />}
                    {item.label}
                  </a>
                </SheetClose>
              );
            })}
          </nav>
        </div>
      </SheetContent>
    </Sheet>
  );
}
