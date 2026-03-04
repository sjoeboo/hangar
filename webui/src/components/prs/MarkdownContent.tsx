import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { cn } from '@/lib/utils'

interface MarkdownContentProps {
  children: string
  className?: string
}

/**
 * Renders GFM markdown with consistent dark-theme styling.
 * Uses react-markdown component overrides instead of the typography plugin
 * so styling works without @tailwindcss/typography installed.
 */
export function MarkdownContent({ children, className }: MarkdownContentProps) {
  return (
    <div className={cn('text-sm leading-relaxed', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          h1: ({ children }) => (
            <h1 className="text-base font-bold text-foreground mt-4 mb-2 pb-1 border-b border-border">
              {children}
            </h1>
          ),
          h2: ({ children }) => (
            <h2 className="text-sm font-semibold text-foreground mt-4 mb-1.5 pb-0.5 border-b border-border/50">
              {children}
            </h2>
          ),
          h3: ({ children }) => (
            <h3 className="text-xs font-semibold text-foreground/80 uppercase tracking-wide mt-3 mb-1">
              {children}
            </h3>
          ),
          p: ({ children }) => (
            <p className="text-foreground/90 mb-2 last:mb-0">{children}</p>
          ),
          ul: ({ children }) => (
            <ul className="list-disc pl-5 mb-2 space-y-0.5 text-foreground/90">{children}</ul>
          ),
          ol: ({ children }) => (
            <ol className="list-decimal pl-5 mb-2 space-y-0.5 text-foreground/90">{children}</ol>
          ),
          li: ({ children }) => <li className="leading-relaxed">{children}</li>,
          // Inline code — overridden for block context via pre below.
          code: ({ children, className: cname }) => (
            <code className={cn(
              'font-mono text-xs bg-muted/80 px-1 py-0.5 rounded text-(--oasis-accent)',
              cname,
            )}>
              {children}
            </code>
          ),
          // Code block: reset the inline code styling set above.
          pre: ({ children }) => (
            <pre className="bg-muted rounded-md p-3 overflow-x-auto mb-2 text-xs [&_code]:bg-transparent [&_code]:p-0 [&_code]:text-foreground/80 [&_code]:rounded-none">
              {children}
            </pre>
          ),
          blockquote: ({ children }) => (
            <blockquote className="border-l-2 border-(--oasis-accent)/40 pl-3 text-foreground/70 italic mb-2">
              {children}
            </blockquote>
          ),
          a: ({ href, children }) => (
            <a
              href={href}
              className="text-(--oasis-accent) underline underline-offset-2 hover:opacity-80"
              target="_blank"
              rel="noopener noreferrer"
            >
              {children}
            </a>
          ),
          strong: ({ children }) => (
            <strong className="font-semibold text-foreground">{children}</strong>
          ),
          em: ({ children }) => (
            <em className="italic text-foreground/80">{children}</em>
          ),
          hr: () => <hr className="border-border my-3" />,
          // GFM tables
          table: ({ children }) => (
            <div className="overflow-x-auto mb-2">
              <table className="text-xs border-collapse w-full">{children}</table>
            </div>
          ),
          th: ({ children }) => (
            <th className="border border-border px-2 py-1 text-left font-semibold bg-muted/60 text-foreground">
              {children}
            </th>
          ),
          td: ({ children }) => (
            <td className="border border-border px-2 py-1 text-foreground/80">{children}</td>
          ),
          // GFM task list checkboxes
          input: ({ type, checked }) =>
            type === 'checkbox' ? (
              <input
                type="checkbox"
                checked={checked}
                readOnly
                className="mr-1.5 accent-(--oasis-accent)"
              />
            ) : null,
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  )
}
