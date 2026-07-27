import { useState, useCallback, useRef, useEffect } from "react";

export interface SlashCommand {
  command: string;
  description: string;
  action: (input: string) => string;
}

const DEFAULT_COMMANDS: SlashCommand[] = [
  { command: "/help",    description: "Show available commands",           action: () => "/help" },
  { command: "/context", description: "Show repo context",                 action: () => "/context" },
  { command: "/clear",   description: "Clear current conversation",        action: () => "/clear" },
  { command: "/model",   description: "Switch model: /model <name>",      action: (input) => input },
  { command: "/review",  description: "Review current branch diff",        action: () => "/review" },
];

export interface SlashCommandState {
  showHints: boolean;
  filteredCommands: SlashCommand[];
  activeIndex: number;
}

export function useSlashCommands(
  input: string,
  onExecute: (command: SlashCommand) => void,
  customCommands?: SlashCommand[],
) {
  const allCommands = customCommands || DEFAULT_COMMANDS;
  const [state, setState] = useState<SlashCommandState>({
    showHints: false,
    filteredCommands: allCommands,
    activeIndex: 0,
  });

  // Detect slash at input start
  useEffect(() => {
    if (input === "/") {
      setState({
        showHints: true,
        filteredCommands: allCommands,
        activeIndex: 0,
      });
    } else if (input.startsWith("/") && state.showHints) {
      const query = input.slice(1).toLowerCase();
      const filtered = allCommands.filter(
        (c) => c.command.toLowerCase().includes(query),
      );
      setState((prev) => ({
        ...prev,
        filteredCommands: filtered,
        activeIndex: 0,
      }));
    } else if (input !== "" && !input.startsWith("/") && state.showHints) {
      setState((prev) => ({ ...prev, showHints: false }));
    }
  }, [input, allCommands, state.showHints]);

  const handleSlashKeyDown = useCallback(
    (e: React.KeyboardEvent): boolean => {
      if (!state.showHints) return false;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setState((prev) => ({
            ...prev,
            activeIndex: Math.min(prev.activeIndex + 1, prev.filteredCommands.length - 1),
          }));
          return true;

        case "ArrowUp":
          e.preventDefault();
          setState((prev) => ({
            ...prev,
            activeIndex: Math.max(prev.activeIndex - 1, 0),
          }));
          return true;

        case "Tab":
        case "Enter":
          if (state.filteredCommands[state.activeIndex]) {
            e.preventDefault();
            const cmd = state.filteredCommands[state.activeIndex];
            onExecute(cmd);
            setState({
              showHints: false,
              filteredCommands: allCommands,
              activeIndex: 0,
            });
            return true;
          }
          return false;

        case "Escape":
          e.preventDefault();
          setState({
            showHints: false,
            filteredCommands: allCommands,
            activeIndex: 0,
          });
          return true;

        default:
          return false;
      }
    },
    [state, allCommands, onExecute],
  );

  const dismissHints = useCallback(() => {
    setState({
      showHints: false,
      filteredCommands: allCommands,
      activeIndex: 0,
    });
  }, [allCommands]);

  return {
    showHints: state.showHints,
    filteredCommands: state.filteredCommands,
    activeIndex: state.activeIndex,
    handleSlashKeyDown,
    dismissHints,
  };
}
