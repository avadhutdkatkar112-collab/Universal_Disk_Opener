import { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('ErrorBoundary caught:', error.message, error.stack, info.componentStack);
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;
      return (
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100%',
          padding: 32,
          color: 'var(--win-text)',
          background: 'var(--win-bg)',
        }}>
          <div style={{
            width: 48,
            height: 48,
            borderRadius: 'var(--win-radius)',
            background: 'rgba(196, 43, 28, 0.1)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginBottom: 16,
          }}>
            <svg width="24" height="24" viewBox="0 0 16 16" fill="none">
              <circle cx="8" cy="8" r="7" stroke="#C42B1C" strokeWidth="1.5" />
              <path d="M8 4.5V8.5" stroke="#C42B1C" strokeWidth="1.5" strokeLinecap="round" />
              <circle cx="8" cy="11" r="0.75" fill="#C42B1C" />
            </svg>
          </div>
          <h2 style={{ fontSize: 16, fontWeight: 600, marginBottom: 4 }}>Something went wrong</h2>
          <p style={{ fontSize: 11, color: 'var(--win-text-secondary)', marginBottom: 16, textAlign: 'center', maxWidth: 400, wordBreak: 'break-all', fontFamily: 'var(--win-font-mono)' }}>
            {this.state.error?.message || 'An unexpected error occurred'}
          </p>
          <pre style={{ fontSize: 10, color: 'var(--win-text-tertiary)', maxWidth: 600, overflow: 'auto', maxHeight: 200, padding: 8, background: 'var(--win-subtle)', borderRadius: 4, whiteSpace: 'pre-wrap' }}>
            {this.state.error?.stack || 'No stack trace'}
          </pre>
          <button
            onClick={() => this.setState({ hasError: false, error: null })}
            style={{
              padding: '6px 16px',
              borderRadius: 'var(--win-radius-sm)',
              background: 'var(--win-accent-default)',
              color: '#fff',
              fontSize: 13,
              fontWeight: 500,
            }}
          >
            Try again
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
