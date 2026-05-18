import { useState, useEffect } from 'react';
import { Save, AlertCircle, CheckCircle2, FileCode, Server, Database, Key } from 'lucide-react';
import api from '@/api/client';
import { cn } from '@/lib/utils';

export default function ConfigPage() {
  const [composeContent, setComposeContent] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error', text: string } | null>(null);
  const [activeTab, setActiveTab] = useState<'infrastructure' | 'providers' | 'advanced'>('infrastructure');

  useEffect(() => {
    fetchCompose();
  }, []);

  const fetchCompose = async () => {
    setIsLoading(true);
    try {
      const response = await api.get('/config/compose');
      setComposeContent(response.data);
    } catch (error) {
      console.error('Failed to fetch compose', error);
      setMessage({ type: 'error', text: 'Failed to load docker-compose.yaml' });
    } finally {
      setIsLoading(false);
    }
  };

  const saveCompose = async () => {
    setIsSaving(true);
    setMessage(null);
    try {
      await api.put('/config/compose', { content: composeContent });
      setMessage({ type: 'success', text: 'docker-compose.yaml saved successfully!' });
    } catch (error) {
      console.error('Failed to save compose', error);
      setMessage({ type: 'error', text: 'Failed to save docker-compose.yaml' });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="p-6 max-w-[1600px] mx-auto space-y-6">
      <div>
        <h1 className="text-xl font-bold text-foreground">System Configuration</h1>
        <p className="text-sm text-muted-foreground">Manage engine configuration, infrastructure definition, and provider settings.</p>
      </div>

      <div className="flex flex-col lg:flex-row gap-6">
         <div className="w-full lg:w-64 shrink-0 space-y-1">
            <button 
              onClick={() => setActiveTab('infrastructure')}
              className={cn("w-full flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-md transition-colors", activeTab === 'infrastructure' ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:bg-secondary hover:text-foreground")}
            >
              <Server size={16} /> Infrastructure (Docker)
            </button>
            <button 
              onClick={() => setActiveTab('providers')}
              className={cn("w-full flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-md transition-colors", activeTab === 'providers' ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:bg-secondary hover:text-foreground")}
            >
              <Key size={16} /> Provider Credentials
            </button>
            <button 
              onClick={() => setActiveTab('advanced')}
              className={cn("w-full flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-md transition-colors", activeTab === 'advanced' ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:bg-secondary hover:text-foreground")}
            >
              <Database size={16} /> Advanced Settings
            </button>
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
                        <div className={`flex items-center gap-1.5 text-xs font-medium ${message.type === 'success' ? 'text-green-600' : 'text-red-600'}`}>
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
                         <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary"></div>
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
                    Editing this file triggers an update to the local deployment. To fully apply network or volume changes, run <code className="bg-blue-500/20 px-1 py-0.5 rounded border border-blue-500/30">docker-compose up -d</code> manually if automatic restart fails.
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'providers' && (
              <div className="bg-card border border-border rounded-lg shadow-sm p-6 text-center text-muted-foreground">
                 <Key size={32} className="mx-auto mb-4 opacity-20" />
                 <h3 className="text-sm font-semibold text-foreground mb-1">Provider Credentials</h3>
                 <p className="text-xs mb-4">Manage API keys and access limits for AI providers.</p>
                 <p className="text-xs italic bg-secondary p-3 rounded">Backend API for secret management is under development.</p>
              </div>
            )}

            {activeTab === 'advanced' && (
              <div className="bg-card border border-border rounded-lg shadow-sm p-6 text-center text-muted-foreground">
                 <Database size={32} className="mx-auto mb-4 opacity-20" />
                 <h3 className="text-sm font-semibold text-foreground mb-1">Advanced Engine Settings</h3>
                 <p className="text-xs mb-4">Configure queue limits, garbage collection, and telemetry settings.</p>
                 <p className="text-xs italic bg-secondary p-3 rounded">Advanced configuration options will be available in the next release.</p>
              </div>
            )}
         </div>
      </div>
    </div>
  );
}
