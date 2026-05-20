import React, { useState, useEffect } from 'react';
import { Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  GitBranch,
  Settings,
  Activity,
  Terminal,
  Database,
  Users,
  HardDrive,
  ListTodo,
  Sun,
  Moon,
  Coins,
  ChevronDown,
  ChevronRight,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { useWebSocket } from '@/hooks/useWebSocket';
import { WsContext } from '@/context/WsContext';
import { useStorageMode } from '@/hooks/useStorageMode';
import { useTheme } from '@/hooks/useTheme';
import { InMemoryBanner } from '@/components/ui/InMemoryBanner';
import { toast } from '@/store/useToastStore';
import api from '@/api/client';

interface SidebarItemProps {
  icon: React.ReactNode;
  label: string;
  to: string;
  active?: boolean;
}

const SidebarItem = ({ icon, label, to, active }: SidebarItemProps) => (
  <Link
    to={to}
    className={cn(
      'flex items-center gap-2.5 px-3 py-2 rounded-md transition-colors text-sm',
      active
        ? 'bg-secondary text-secondary-foreground font-medium'
        : 'text-muted-foreground hover:bg-secondary/50 hover:text-foreground font-medium'
    )}
  >
    {icon}
    <span>{label}</span>
  </Link>
);

interface ModelTokenStats {
  model: string;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
}

interface TokenUsage {
  totalPromptTokens: number;
  totalCompletionTokens: number;
  totalTokens: number;
  byModel: ModelTokenStats[];
}

function useTokenUsage() {
  const [usage, setUsage] = useState<TokenUsage | null>(null);

  useEffect(() => {
    const fetch = async () => {
      try {
        const res = await api.get('/metrics');
        const data = res.data as { tokenUsage?: TokenUsage };
        if (data.tokenUsage) setUsage(data.tokenUsage);
      } catch { /* silent */ }
    };
    fetch();
    const id = setInterval(fetch, 10_000);
    return () => clearInterval(id);
  }, []);

  return usage;
}

function fmt(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
  return String(n);
}

function TokenUsagePanel() {
  const usage = useTokenUsage();
  const [expanded, setExpanded] = useState(false);

  if (!usage || usage.totalTokens === 0) return null;

  return (
    <div className="rounded-md border border-border bg-muted/30 overflow-hidden text-xs">
      <button
        onClick={() => setExpanded((e) => !e)}
        className="w-full flex items-center justify-between px-2.5 py-2 hover:bg-muted/50 transition-colors"
      >
        <div className="flex items-center gap-1.5 text-muted-foreground font-medium">
          <Coins size={12} className="text-amber-500" />
          <span>Tokens used</span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="font-bold text-foreground">{fmt(usage.totalTokens)}</span>
          {expanded ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
        </div>
      </button>

      {expanded && (
        <div className="border-t border-border px-2.5 py-2 space-y-2">
          <div className="flex justify-between text-muted-foreground">
            <span>Prompt</span>
            <span className="font-medium text-foreground">{fmt(usage.totalPromptTokens)}</span>
          </div>
          <div className="flex justify-between text-muted-foreground">
            <span>Completion</span>
            <span className="font-medium text-foreground">{fmt(usage.totalCompletionTokens)}</span>
          </div>
          {usage.byModel && usage.byModel.length > 0 && (
            <div className="pt-1 border-t border-border/50 space-y-1">
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold">By model</p>
              {usage.byModel.map((m) => (
                <div key={m.model} className="flex justify-between items-center gap-1">
                  <span className="truncate text-muted-foreground max-w-[120px]" title={m.model}>
                    {m.model}
                  </span>
                  <span className="font-medium text-foreground shrink-0">{fmt(m.totalTokens)}</span>
                </div>
              ))}
            </div>
          )}
          <p className="text-[10px] text-muted-foreground/60 pt-0.5">Session only · resets on restart</p>
        </div>
      )}
    </div>
  );
}

export const DashboardLayout = ({ children }: { children: React.ReactNode }) => {
  const location = useLocation();
  const { isConnected, events, addListener } = useWebSocket();
  const storageMode = useStorageMode();
  const { theme, toggleTheme } = useTheme();

  const primaryItems = [
    { icon: <LayoutDashboard size={16} />, label: 'Dashboard', to: '/' },
    { icon: <GitBranch size={16} />, label: 'Workflows', to: '/workflows' },
    { icon: <Activity size={16} />, label: 'Executions', to: '/executions' },
  ];

  const secondaryItems = [
    { icon: <ListTodo size={16} />, label: 'Queues & Tasks', to: '/queues' },
    { icon: <Users size={16} />, label: 'Agents', to: '/agents' },
    { icon: <HardDrive size={16} />, label: 'Providers', to: '/providers' },
    { icon: <Database size={16} />, label: 'Artifacts & Memory', to: '/artifacts' },
    { icon: <Settings size={16} />, label: 'Settings', to: '/config' },
  ];

  React.useEffect(() => {
    return addListener((e) => {
      switch (e.type) {
        case 'WORKFLOW_STARTED':
          toast.info(`Workflow execution started`);
          break;
        case 'WORKFLOW_COMPLETED':
          toast.success(`Workflow completed successfully`);
          break;
        case 'WORKFLOW_FAILED':
          const err = (e.payload as any)?.error || 'Unknown error';
          toast.error(`Workflow failed: ${err}`);
          break;
        case 'TASK_FAILED':
          const terr = (e.payload as any)?.error || 'Unknown error';
          toast.error(`Task ${e.taskId} failed: ${terr}`);
          break;
        case 'TASK_WAITING_FOR_APPROVAL':
          toast.warning(`Task ${e.taskId} requires manual approval`);
          break;
      }
    });
  }, [addListener]);

  const allItems = [...primaryItems, ...secondaryItems];

  return (
    <div className="flex h-screen bg-muted/20 overflow-hidden text-sm">
      {/* Sidebar */}
      <aside className="w-[240px] border-r border-border bg-card flex flex-col shrink-0">
        <div className="h-12 border-b border-border flex items-center px-4">
          <div className="flex items-center gap-2 font-bold text-foreground">
            <div className="w-6 h-6 bg-primary rounded flex items-center justify-center text-primary-foreground">
              <Terminal size={14} />
            </div>
            <span>LLM Orchestrator</span>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto py-3 px-3 space-y-6">
          <div>
            <div className="px-2 mb-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider">Core</div>
            <nav className="space-y-0.5">
              {primaryItems.map((item) => (
                <SidebarItem
                  key={item.to}
                  {...item}
                  active={location.pathname === item.to || (item.to !== '/' && location.pathname.startsWith(item.to))}
                />
              ))}
            </nav>
          </div>

          <div>
            <div className="px-2 mb-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider">Resources</div>
            <nav className="space-y-0.5">
              {secondaryItems.map((item) => (
                <SidebarItem
                  key={item.to}
                  {...item}
                  active={location.pathname === item.to || (item.to !== '/' && location.pathname.startsWith(item.to))}
                />
              ))}
            </nav>
          </div>
        </div>

        <div className="p-3 border-t border-border mt-auto space-y-2">
          {/* Token usage panel */}
          <TokenUsagePanel />

          <div
            className={cn(
              'flex items-center gap-2 px-2 py-1.5 rounded border text-xs font-medium',
              isConnected
                ? 'bg-green-500/10 border-green-500/20 text-green-700'
                : 'bg-red-500/10 border-red-500/20 text-red-600'
            )}
          >
            <div
              className={cn(
                'w-1.5 h-1.5 rounded-full',
                isConnected ? 'bg-green-500 animate-pulse' : 'bg-red-500'
              )}
            />
            <span>{isConnected ? 'Engine Connected' : 'Engine Disconnected'}</span>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 flex flex-col min-w-0">
        <header className="h-12 border-b border-border bg-card flex items-center justify-between px-4 shrink-0">
          <div className="flex items-center gap-2 text-sm">
            <span className="font-semibold text-foreground">
              {allItems.find(
                (item) =>
                  location.pathname === item.to ||
                  (item.to !== '/' && location.pathname.startsWith(item.to))
              )?.label || 'Dashboard'}
            </span>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={toggleTheme}
              className="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors border border-border"
              title={theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}
            >
              {theme === 'light' ? <Moon size={16} /> : <Sun size={16} />}
            </button>
            <div
              className={cn(
                'flex items-center gap-1.5 px-2 py-1 rounded border text-[10px] font-semibold',
                isConnected
                  ? 'bg-green-500/10 border-green-500/20 text-green-600'
                  : 'bg-red-500/10 border-red-500/20 text-red-600'
              )}
            >
              <div
                className={cn(
                  'w-1 h-1 rounded-full',
                  isConnected ? 'bg-green-500 animate-pulse' : 'bg-red-500'
                )}
              />
              {isConnected ? 'WS LIVE' : 'WS OFFLINE'}
            </div>
          </div>
        </header>

        <div className="flex-1 overflow-auto bg-muted/20">
          <WsContext.Provider value={{ isConnected, events, addListener }}>
            {storageMode === 'memory' && <InMemoryBanner />}
            {children}
          </WsContext.Provider>
        </div>

      </main>
    </div>
  );
};

