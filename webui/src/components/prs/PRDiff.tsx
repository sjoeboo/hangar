import { useState } from 'react'
import type { PRFileChange, PRReview } from '@/api/types'
import { cn } from '@/lib/utils'

interface PRDiffProps {
  files: PRFileChange[]
  diffContent: string
  reviews: PRReview[]
}

function parseHunks(diff: string): Map<string, string> {
  // Split diff into per-file sections: "diff --git a/... b/..."
  const fileMap = new Map<string, string>()
  const sections = diff.split(/^diff --git /m).filter(Boolean)
  for (const section of sections) {
    const firstLine = section.split('\n')[0]
    // Extract "b/<path>" filename
    const match = firstLine.match(/b\/(.+)$/)
    if (match) {
      fileMap.set(match[1], section)
    }
  }
  return fileMap
}

function DiffLine({ line }: { line: string }) {
  if (line.startsWith('@@')) {
    return (
      <div className="px-2 py-0.5 text-xs font-mono text-blue-400 bg-blue-400/5 select-none">
        {line}
      </div>
    )
  }
  if (line.startsWith('+')) {
    return (
      <div className="px-2 py-0.5 text-xs font-mono text-green-400 bg-green-400/5 whitespace-pre">
        {line}
      </div>
    )
  }
  if (line.startsWith('-')) {
    return (
      <div className="px-2 py-0.5 text-xs font-mono text-red-400 bg-red-400/5 whitespace-pre">
        {line}
      </div>
    )
  }
  return (
    <div className="px-2 py-0.5 text-xs font-mono text-foreground/70 whitespace-pre">
      {line}
    </div>
  )
}

const MAX_DIFF_LINES = 300

function TruncatedDiff({ fileDiff }: { fileDiff: string }) {
  const [showAll, setShowAll] = useState(false)
  const allLines = fileDiff.split('\n').slice(4)
  const truncated = !showAll && allLines.length > MAX_DIFF_LINES
  const lines = truncated ? allLines.slice(0, MAX_DIFF_LINES) : allLines
  return (
    <>
      {lines.map((line, i) => (
        <DiffLine key={i} line={line} />
      ))}
      {truncated && (
        <button
          onClick={() => setShowAll(true)}
          className="w-full px-3 py-2 text-xs text-(--oasis-accent) hover:bg-muted/50 transition-colors text-left"
        >
          … {allLines.length - MAX_DIFF_LINES} more lines — click to show all
        </button>
      )}
    </>
  )
}

function getInlineComments(filePath: string, reviews: PRReview[]) {
  const comments: Array<{ author: string; body: string; line?: number }> = []
  for (const review of reviews) {
    for (const comment of review.comments) {
      if (comment.path === filePath) {
        comments.push({ author: review.author, body: comment.body, line: comment.line })
      }
    }
  }
  return comments
}

function FileDiff({ file, hunkMap, reviews, defaultExpanded }: {
  file: PRFileChange
  hunkMap: Map<string, string>
  reviews: PRReview[]
  defaultExpanded?: boolean
}) {
  const [expanded, setExpanded] = useState(defaultExpanded ?? false)
  const fileDiff = hunkMap.get(file.path)
  const inlineComments = getInlineComments(file.path, reviews)

  const statusColor: Record<string, string> = {
    added:    'text-(--oasis-green)',
    removed:  'text-(--oasis-red)',
    modified: 'text-(--oasis-yellow)',
    renamed:  'text-blue-400',
  }

  return (
    <div className="border border-border rounded-lg overflow-hidden mb-3">
      {/* File header */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-2 px-3 py-2 bg-muted/50 hover:bg-muted transition-colors text-left"
      >
        <span className="text-[10px] mr-1 text-muted-foreground">{expanded ? '▼' : '▶'}</span>
        <span className={cn('text-xs font-mono font-medium flex-1 truncate', statusColor[file.status] ?? 'text-foreground')}>
          {file.path}
        </span>
        <span className="shrink-0 text-xs text-muted-foreground">
          <span className="text-green-400">+{file.additions}</span>
          {' '}
          <span className="text-red-400">-{file.deletions}</span>
        </span>
      </button>

      {/* Diff content */}
      {expanded && (
        <div className="bg-background">
          {fileDiff ? (
            <TruncatedDiff fileDiff={fileDiff} />
          ) : (
            <div className="px-3 py-2 text-xs text-muted-foreground italic">
              Diff not available for this file
            </div>
          )}
          {/* Inline comments */}
          {inlineComments.length > 0 && (
            <div className="border-t border-border bg-muted/30 p-3 space-y-2">
              {inlineComments.map((c, i) => (
                <div key={i} className="flex gap-2 text-xs">
                  <span className="font-medium text-foreground shrink-0">{c.author}</span>
                  {c.line && <span className="text-blue-400 font-mono shrink-0">:{c.line}</span>}
                  <span className="text-foreground/80">{c.body}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export function PRDiff({ files, diffContent, reviews }: PRDiffProps) {
  if (files.length === 0) {
    return (
      <div className="flex items-center justify-center py-12 text-muted-foreground text-sm">
        No file changes
      </div>
    )
  }

  const totalAdditions = files.reduce((s, f) => s + f.additions, 0)
  const totalDeletions = files.reduce((s, f) => s + f.deletions, 0)
  const hunkMap = parseHunks(diffContent)

  return (
    <div className="py-2">
      {/* Summary bar */}
      <div className="mb-3 text-xs text-muted-foreground flex items-center gap-3">
        <span>{files.length} file{files.length !== 1 ? 's' : ''} changed</span>
        <span className="text-green-400">+{totalAdditions}</span>
        <span className="text-red-400">-{totalDeletions}</span>
      </div>

      {/* File diffs */}
      {files.map((file) => (
        <FileDiff
          key={file.path}
          file={file}
          hunkMap={hunkMap}
          reviews={reviews}
          defaultExpanded={files.length === 1}
        />
      ))}
    </div>
  )
}
