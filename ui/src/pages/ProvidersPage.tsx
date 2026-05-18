import { useWorkflowStore } from '@/store/useWorkflowStore';
import { HardDrive, Cpu, Globe, Loader2, AlertCircle } from 'lucide-react';
import { useEffect } from 'react';

export default function ProvidersPage() {
  const { providers, fetchProviders } = useWorkflowStore();

  useEffect(() => {
    fetchProviders();
  }, [fetchProviders]);

  if (providers.isLoading) {
    return (
      <div className="p-6 max-w-[1600px] mx-auto flex justify-center items-center h-[50vh]">
        <Loader2 className="animate-spin h-8 w-8 text-primary" />
      </div>
    );
  }

  if (providers.error) {
    return (
      <div className="p-6 max-w-[1600px] mx-auto">
        <div className="bg-red-500/10 border border-red-500/20 text-red-500 p-4 rounded-lg">
          <h2 className="font-bold mb-1">Failed to Load Providers</h2>
          <p className="text-sm font-mono">{providers.error}</p>
          <button onClick={fetchProviders} className="mt-3 text-xs font-medium text-red-600 hover:underline">
            Retry →
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-[1600px] mx-auto space-y-6">
      <div>
        <h1 className="text-xl font-bold text-foreground">AI Infrastructure Providers</h1>
        <p className="text-sm text-muted-foreground">
          Providers with a configured API key are shown here. Models are fetched live from each provider's API.
        </p>
      </div>

      {providers.data.length === 0 ? (
        <div className="col-span-full py-20 text-center bg-card border border-dashed border-border rounded-lg">
          <HardDrive size={48} className="mx-auto text-muted-foreground opacity-20 mb-4" />
          <p className="text-muted-foreground font-medium">No active providers detected.</p>
          <p className="text-xs text-muted-foreground mt-2 max-w-sm mx-auto">
            Set a provider API key in your environment (e.g.{' '}
            <code className="bg-secondary px-1 rounded">GROQ_API_KEY</code>,{' '}
            <code className="bg-secondary px-1 rounded">OPENAI_API_KEY</code>) and restart the server.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {providers.data.map((p) => (
            <div key={p.name} className="bg-card border border-border rounded-lg shadow-sm flex flex-col overflow-hidden">
              <div className="p-4 border-b border-border bg-muted/30 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 bg-primary/10 rounded-md flex items-center justify-center text-primary">
                    <Globe size={18} />
                  </div>
                  <h3 className="font-bold text-lg capitalize">{p.name}</h3>
                </div>
                <div className="flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-secondary border border-border text-[10px] font-bold text-muted-foreground">
                  API KEY SET
                </div>
              </div>

              <div className="p-4 flex-1 space-y-4">
                <div>
                  <div className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest mb-2 flex items-center gap-1.5">
                    <Cpu size={12} /> Available Models ({p.models.length})
                  </div>
                  {p.models.length === 0 ? (
                    <div className="flex items-center gap-2 text-xs text-amber-600 bg-amber-500/10 border border-amber-500/20 rounded p-2">
                      <AlertCircle size={12} />
                      Could not retrieve models from this provider.
                    </div>
                  ) : (
                    <div className="flex flex-wrap gap-2">
                      {p.models.map((m) => (
                        <span
                          key={m}
                          className="px-2 py-1 bg-secondary text-secondary-foreground text-xs font-medium rounded border border-border"
                        >
                          {m}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
