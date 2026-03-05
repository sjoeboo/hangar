import { Component, type ReactNode } from 'react'

interface Props {
  children: ReactNode
  fallback?: (error: Error) => ReactNode
}

interface State {
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { error: null }
  }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  override render() {
    if (this.state.error) {
      if (this.props.fallback) return this.props.fallback(this.state.error)
      return (
        <div className="flex flex-col items-center justify-center py-12 gap-3 text-sm">
          <span className="text-2xl opacity-40">⚠</span>
          <p className="text-(--oasis-red) font-medium">Failed to render</p>
          <p className="text-muted-foreground text-xs font-mono max-w-sm text-center break-all">
            {this.state.error.message}
          </p>
        </div>
      )
    }
    return this.props.children
  }
}
