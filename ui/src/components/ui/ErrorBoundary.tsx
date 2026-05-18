import React from 'react';
import { AlertTriangle, RefreshCw } from 'lucide-react';

interface Props {
  children: React.ReactNode;
  fallbackTitle?: string;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error('[ErrorBoundary] Uncaught error:', error, info.componentStack);
  }

  render() {
    if (!this.state.hasError) return this.props.children;

    return (
      <div className="h-full flex items-center justify-center p-8">
        <div className="max-w-lg w-full bg-card border border-red-500/20 rounded-xl shadow-sm p-8 text-center space-y-4">
          <div className="w-14 h-14 bg-red-500/10 rounded-full flex items-center justify-center mx-auto">
            <AlertTriangle className="text-red-500" size={28} />
          </div>
          <div>
            <h2 className="text-lg font-bold text-foreground mb-1">
              {this.props.fallbackTitle ?? 'Something went wrong'}
            </h2>
            <p className="text-sm text-muted-foreground">
              An unexpected error occurred in this component. You can try refreshing the page, or navigate
              to another section.
            </p>
          </div>
          {this.state.error && (
            <pre className="text-left text-[11px] bg-muted/50 text-muted-foreground p-3 rounded border border-border overflow-auto max-h-32">
              {this.state.error.message}
            </pre>
          )}
          <button
            onClick={() => window.location.reload()}
            className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md text-sm font-medium hover:opacity-90 transition-opacity"
          >
            <RefreshCw size={14} />
            Reload Page
          </button>
        </div>
      </div>
    );
  }
}
