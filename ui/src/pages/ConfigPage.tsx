import { useState, useEffect } from 'react';
import { Save, AlertCircle, CheckCircle2, FileCode, Server, Key, Database } from 'lucide-react';
import api from '@/api/client';
import { cn } from '@/lib/utils';

type Tab = 'infrastructure';

export default function ConfigPage() {
  const [composeContent, setComposeContent] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [activeTab] = useState<Tab>('infrastructure');

  useEffect(() => {
    fetchCompose();
  }, []);

  const fetchCompose = async () => {
    setIsLoading(true);
    setMessage(null);
    try {
      const response = await api.get('/config/compose');
      // Backend returns plain text for this endpoint
      setComposeContent(typeof response.data === 'string' ? response.data : JSON.stringify(response.data, null, 2));
    } catch (error: unknown) {
      let msg = 'Failed to load docker-compose.yaml';
      const err = error as Record<string, unknown>;
      if (err.response && typeof err.response === 'object') {
        const resp = err.response as Record<string, unknown>;
        if (typeof resp.data === 'string') msg = resp.data;
      }
      setMessage({ type: 'error', text: msg });
    } finally {
      setIsLoading(false);
    }
  };

  const saveCompose = async () => {
    if (!composeContent.trim()) {
      setMessage({ type: 'error', text: 'Content cannot be empty.' });
      return;
    }
    setIsSaving(true);
    setMessage(null);
    try {
      await api.put('/config/compose', { content: composeContent });
      setMessage({ type: 'success', text: 'docker-compose.yaml saved successfully!' });
      setTimeout(() => setMessage(null), 5000);
    } catch (error: unknown) {
      let msg = 'Failed to save docker-compose.yaml';
      const err = error as Record<string, unknown>;
      if (err.response && typeof err.response === 'object') {
        const resp = err.response as Record<string, unknown>;
        if (resp.data && typeof resp.data === 'object') {
          const data = resp.data as Record<string, string>;
          if (data.error) msg = data.error;
        }
      }
      setMessage({ type: 'error', text: msg });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="p-6 max-w-[1600px] mx-auto space-y-6">
      <div>
        <h1 className="text-xl font-bold text-foreground">System Configuration</h1>
        <p className="text-sm text-muted-foreground">
          Manage engine configuration, infrastructure definition, and provider settings.
        </p>
      </div>

      <div className="flex flex-col lg:flex-row gap-6">
        <div className="w-full lg:w-64 shrink-0 space-y-1">
          <button
            className={cn(
              'w-full flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-md transition-colors',
              activeTab === 'infrastructure'
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
            )}
          >
            <Server size={16} /> Infrastructure (Docker)
          </button>
          <div className="px-3 py-2 text-sm font-medium rounded-md text-muted-foreground opacity-50 cursor-not-allowed flex items-center gap-2">
            <Key size={16} />
            <div>
              <div>Provider Credentials</div>
              <div className="text-[10px] text-muted-foreground">Set via environment variables</div>
            </div>
          </div>
          <div className="px-3 py-2 text-sm font-medium rounded-md text-muted-foreground opacity-50 cursor-not-allowed flex items-center gap-2">
            <Database size={16} />
            <div>
              <div>Advanced Settings</div>
              <div className="text-[10px] text-muted-foreground">Coming soon</div>
            </div>
          </div>
        </div>

        <div className="flex-1">
          {activeTab === 'infrastructure' && (
            <div className="space-y-4">
              <div className="bg-card border border-border rounded-lg shadow-sm overflow-hidden flex flex-col h-[700px]">
                <div className="bg-secondary/50 px-4 py-2.5 border-b border-border flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <FileCode size={16} className="text-muted-foreground" />
                    <span className="font-mono text-xs font-semibold text-foreground">docker-compose.yaml</span>
                  </div>
                  <div className="flex items-center gap-3">
                    {message && (
                      <div
                        className={cn(
                          'flex items-center gap-1.5 text-xs font-medium',
                          message.type === 'success' ? 'text-green-600' : 'text-red-600'
                        )}
                      >
                        {message.type === 'success' ? <CheckCircle2 size={14} /> : <AlertCircle size={14} />}
                        {message.text}
                      </div>
                    )}
                    <button
                      onClick={saveCompose}
                      disabled={isSaving || isLoading}
                      className="flex items-center gap-1.5 bg-primary text-primary-foreground px-3 py-1.5 rounded text-xs font-medium hover:opacity-90 transition-opacity disabled:opacity-50"
                    >
                      <Save size={14} />
                      {isSaving ? 'Saving...' : 'Apply Changes'}
                    </button>
                  </div>
                </div>

                <div className="flex-1 relative bg-[#0d1117]">
                  {isLoading ? (
                    <div className="absolute inset-0 flex items-center justify-center">
                      <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary" />
                    </div>
                  ) : (
                    <textarea
                      value={composeContent}
                      onChange={(e) => setComposeContent(e.target.value)}
                      className="w-full h-full p-4 font-mono text-[13px] bg-transparent text-[#e6edf3] focus:outline-none resize-none leading-relaxed"
                      spellCheck={false}
                    />
                  )}
                </div>
              </div>

              <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-3 flex gap-3">
                <AlertCircle className="text-blue-600 shrink-0 mt-0.5" size={16} />
                <div className="text-xs text-blue-900 dark:text-blue-300">
                  <span className="font-semibold block mb-0.5">Infrastructure State Note</span>
                  Editing this file updates it on disk. To apply network or volume changes, run{' '}
                  <code className="bg-blue-500/20 px-1 py-0.5 rounded border border-blue-500/30">
                    docker-compose up -d
                  </code>{' '}
                  manually after saving.
                  Provider API keys must be set as environment variables — they are not managed through this UI.
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
