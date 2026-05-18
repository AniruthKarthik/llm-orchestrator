import React from 'react';
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
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { useWebSocket } from '@/hooks/useWebSocket';
import { WsContext } from '@/context/WsContext';

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

export const DashboardLayout = ({ children }: { children: React.ReactNode }) => {
  const location = useLocation();
  const { isConnected, events, addListener } = useWebSocket();

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
            <span>Orchestrator Plane</span>
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

        <div className="p-3 border-t border-border mt-auto">
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
            {children}
          </WsContext.Provider>
        </div>

      </main>
    </div>
  );
};
