import { useEffect, useState } from 'react';
import { Database, FileText, Code, Clock, Search, X, Loader2 } from 'lucide-react';
import { useWorkflowStore } from '@/store/useWorkflowStore';
import type { Artifact } from '@/types';

function ArtifactPreviewModal({ artifact, onClose }: { artifact: Artifact; onClose: () => void }) {
  // Intelligently extract actual text response if data is an object or stringified JSON
  let displayContent = artifact.data;
  let isExtracted = false;

  let parsedData = artifact.data;
  
  // If data is a string, it might be stringified JSON from the backend
  if (typeof artifact.data === 'string') {
    try {
      parsedData = JSON.parse(artifact.data);
    } catch (e) {
      // not JSON, keep as string
    }
  }

  if (typeof parsedData === 'object' && parsedData !== null) {
    const d = parsedData as any;
    if (d.response && typeof d.response === 'string') {
      displayContent = d.response;
      isExtracted = true;
    } else if (d.output && typeof d.output === 'string') {
      displayContent = d.output;
      isExtracted = true;
    } else {
      displayContent = parsedData;
    }
  }

  const renderAsJson = !isExtracted && (artifact.type === 'JSON' || typeof displayContent !== 'string');
  const renderAsCode = !isExtracted && artifact.type === 'CODE';

  return (
    <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4" onClick={onClose}>
      <div
        className="bg-card border border-border rounded-lg shadow-2xl w-full max-w-2xl max-h-[80vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between p-4 border-b border-border">
          <div>
            <h2 className="font-bold text-foreground">{artifact.name}</h2>
            <p className="text-xs text-muted-foreground mt-0.5">
              Workflow: {artifact.workflowId} · Task: {artifact.taskId}
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded hover:bg-secondary text-muted-foreground hover:text-foreground"
          >
            <X size={18} />
          </button>
        </div>
        <div className="flex-1 overflow-auto bg-muted/5 p-6">
          <div className="max-w-3xl mx-auto">
            {renderAsJson ? (
              <pre className="text-xs font-mono bg-[#0d1117] text-[#e6edf3] p-4 rounded-md overflow-auto whitespace-pre-wrap break-all shadow-inner border border-[#30363d]">
                {JSON.stringify(displayContent, null, 2)}
              </pre>
            ) : renderAsCode ? (
              <pre className="text-xs font-mono bg-[#0d1117] text-[#e6edf3] p-4 rounded-md overflow-auto whitespace-pre-wrap break-all shadow-inner border border-[#30363d]">
                {displayContent as string}
              </pre>
            ) : (
              <div className="prose prose-sm dark:prose-invert max-w-none">
                <div className="text-[14px] leading-relaxed text-foreground whitespace-pre-wrap font-sans">
                  {displayContent as string}
                </div>
              </div>
            )}
          </div>
        </div>
        <div className="p-3 bg-card border-t border-border text-[11px] text-muted-foreground flex justify-between">
          <span>Created: {new Date(artifact.createdAt).toLocaleString()}</span>
          <span className="uppercase font-semibold tracking-wider">{artifact.type}</span>
        </div>
      </div>
    </div>
  );
}

export default function ArtifactsPage() {
  const { artifacts, fetchArtifacts } = useWorkflowStore();
  const [search, setSearch] = useState('');
  const [preview, setPreview] = useState<Artifact | null>(null);

  useEffect(() => {
    fetchArtifacts();
  }, [fetchArtifacts]);

  const filtered = artifacts.data.filter(
    (a) =>
      a.name?.toLowerCase().includes(search.toLowerCase()) ||
      a.workflowId?.toLowerCase().includes(search.toLowerCase()) ||
      a.type?.toLowerCase().includes(search.toLowerCase())
  );

  if (artifacts.isLoading) {
    return (
      <div className="p-6 max-w-[1600px] mx-auto flex justify-center items-center h-[50vh]">
        <Loader2 className="animate-spin h-8 w-8 text-primary" />
      </div>
    );
  }

  if (artifacts.error) {
    return (
      <div className="p-6 max-w-[1600px] mx-auto">
        <div className="bg-red-500/10 border border-red-500/20 text-red-500 p-4 rounded-lg">
          <h2 className="font-bold mb-1">Failed to Load Artifacts</h2>
          <p className="text-sm font-mono">{artifacts.error}</p>
          <button onClick={fetchArtifacts} className="mt-3 text-xs font-medium text-red-600 hover:underline">
            Retry →
          </button>
        </div>
      </div>
    );
  }

  return (
    <>
      {preview && <ArtifactPreviewModal artifact={preview} onClose={() => setPreview(null)} />}

      <div className="p-6 max-w-[1600px] mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold text-foreground">Artifacts & Memory Explorer</h1>
            <p className="text-sm text-muted-foreground">
              Inspect generated outputs, datasets, and workflow context lineage.
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* Sidebar */}
          <div className="space-y-4">
            <div className="bg-card border border-border rounded-lg p-4 space-y-4 shadow-sm">
              <div>
                <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider mb-2 block">
                  Search Store
                </label>
                <div className="relative">
                  <Search className="absolute left-2.5 top-2 text-muted-foreground" size={12} />
                  <input
                    className="w-full bg-secondary/50 border border-border rounded-md pl-8 pr-2 py-1.5 text-xs outline-none focus:ring-1 focus:ring-primary"
                    placeholder="Name, workflow, or type..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                  />
                </div>
              </div>
            </div>

            <div className="bg-primary/5 border border-primary/20 rounded-lg p-4">
              <div className="flex items-center gap-2 text-primary mb-2">
                <Database size={16} />
                <span className="text-xs font-bold uppercase">Memory Usage</span>
              </div>
              <div className="text-2xl font-bold text-primary">{artifacts.data.length}</div>
              <div className="text-[10px] text-muted-foreground mt-1">
                {filtered.length !== artifacts.data.length
                  ? `${filtered.length} matching filter`
                  : 'Total artifacts stored'}
              </div>
            </div>
          </div>

          {/* Artifact List */}
          <div className="lg:col-span-3 space-y-4">
            <div className="bg-card border border-border rounded-lg shadow-sm overflow-hidden">
              <div className="p-3 border-b border-border bg-muted/20 flex items-center justify-between">
                <span className="text-xs font-bold text-muted-foreground">
                  Artifacts ({filtered.length})
                </span>
              </div>
              <div className="divide-y divide-border">
                {filtered.length === 0 ? (
                  <div className="p-12 text-center text-muted-foreground italic text-sm">
                    {search
                      ? 'No artifacts match your search.'
                      : 'No artifacts have been generated yet. Run a workflow to see results.'}
                  </div>
                ) : (
                  filtered.map((art) => (
                    <div
                      key={art.id}
                      className="p-4 flex items-center justify-between hover:bg-secondary/20 transition-colors group"
                    >
                      <div className="flex items-center gap-4">
                        <div className="w-10 h-10 bg-muted border border-border rounded flex items-center justify-center text-muted-foreground group-hover:bg-primary/10 group-hover:text-primary transition-colors">
                          {art.type === 'TEXT' || art.type === 'JSON' ? (
                            <FileText size={20} />
                          ) : (
                            <Code size={20} />
                          )}
                        </div>
                        <div>
                          <div className="text-sm font-bold text-foreground flex items-center gap-2">
                            {art.name}
                            <span className="text-[10px] bg-secondary px-1 rounded text-muted-foreground uppercase">
                              {art.type}
                            </span>
                          </div>
                          <div className="text-xs text-muted-foreground flex items-center gap-3 mt-0.5">
                            <span className="flex items-center gap-1">
                              <Clock size={12} />
                              {new Date(art.createdAt).toLocaleString()}
                            </span>
                            <span>•</span>
                            <span className="font-medium text-foreground/70 font-mono text-[10px]">
                              {art.workflowId}
                            </span>
                          </div>
                        </div>
                      </div>
                      <button
                        onClick={() => setPreview(art)}
                        className="px-3 py-1.5 bg-primary text-primary-foreground text-xs font-bold rounded-md transition-opacity hover:opacity-90"
                      >
                        Preview
                      </button>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
