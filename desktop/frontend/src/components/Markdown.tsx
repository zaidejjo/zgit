import { memo, type ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { cn } from "@/lib/utils";
import { OpenURL } from "../../wailsjs/go/main/App";

interface MarkdownProps {
  text: string;
  className?: string;
  compact?: boolean;
}

// ExternalLink opens URLs through the Go backend instead of navigating
// the Wails webview away from the app.
function ExternalLink({ href, children }: { href?: string; children?: ReactNode }) {
  if (!href) {
    return <span>{children}</span>;
  }
  return (
    <a
      href={href}
      className="text-primary hover:underline break-all"
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
        OpenURL(href).catch(() => window.open(href, "_blank", "noopener"));
      }}
    >
      {children}
    </a>
  );
}

// Markdown renders GitHub-flavored markdown safely.
// react-markdown escapes raw HTML by default (no rehype-raw), so hostile
// PR/issue bodies cannot inject markup or scripts.
function MarkdownInner({ text, className, compact }: MarkdownProps) {
  return (
    <div className={cn("markdown-body text-sm leading-relaxed", className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          a: ExternalLink,
          h1: ({ children }) => (
            <h1 className={cn("font-bold text-foreground", compact ? "text-sm mt-2 mb-1" : "text-lg mt-3 mb-1.5")}>{children}</h1>
          ),
          h2: ({ children }) => (
            <h2 className={cn("font-bold text-foreground", compact ? "text-sm mt-2 mb-1" : "text-base mt-3 mb-1.5")}>{children}</h2>
          ),
          h3: ({ children }) => (
            <h3 className="font-semibold text-foreground text-sm mt-2 mb-1">{children}</h3>
          ),
          h4: ({ children }) => (
            <h4 className="font-semibold text-foreground text-sm mt-1.5 mb-0.5">{children}</h4>
          ),
          p: ({ children }) => <p className="my-1.5 first:mt-0 last:mb-0">{children}</p>,
          ul: ({ children }) => <ul className="list-disc list-outside ml-5 my-1.5 space-y-0.5">{children}</ul>,
          ol: ({ children }) => <ol className="list-decimal list-outside ml-5 my-1.5 space-y-0.5">{children}</ol>,
          li: ({ children, ...props }) => (
            <li className={cn((props as { className?: string }).className?.includes("task-list-item") && "list-none -ml-5")} {...props}>
              {children}
            </li>
          ),
          blockquote: ({ children }) => (
            <blockquote className="border-l-2 border-primary/40 pl-3 my-1.5 text-muted-foreground">{children}</blockquote>
          ),
          pre: ({ children }) => (
            <pre className="bg-muted/30 rounded-md p-2.5 text-xs font-mono overflow-x-auto border border-border/30 my-1.5">
              {children}
            </pre>
          ),
          code: ({ children, className: codeClass }) => {
            // Fenced blocks render inside <pre>; inline code gets a pill.
            const lang = /language-(\w+)/.exec(codeClass || "")?.[1];
            if (!lang) {
              return (
                <code className="bg-muted/50 rounded px-1 py-0.5 text-xs font-mono break-words">
                  {children}
                </code>
              );
            }
            return (
              <>
                <div className="text-[10px] text-muted-foreground/60 uppercase tracking-wider mb-1 -mt-0.5">
                  {lang}
                </div>
                <code>{children}</code>
              </>
            );
          },
          table: ({ children }) => (
            <div className="overflow-x-auto my-1.5">
              <table className="text-xs border-collapse border border-border/40">{children}</table>
            </div>
          ),
          thead: ({ children }) => <thead className="bg-muted/40">{children}</thead>,
          th: ({ children }) => (
            <th className="border border-border/40 px-2 py-1 text-left font-semibold">{children}</th>
          ),
          td: ({ children }) => <td className="border border-border/40 px-2 py-1">{children}</td>,
          hr: () => <hr className="border-border/40 my-2" />,
          img: ({ src, alt }) => (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={src} alt={alt} className="rounded-md max-w-full my-1.5" loading="lazy" />
          ),
        }}
      >
        {text}
      </ReactMarkdown>
    </div>
  );
}

const Markdown = memo(MarkdownInner);
export default Markdown;
