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
  Search,
  Bell,
  HardDrive,
  ListTodo
} from 'lucide-react';
import { cn } from '@/lib/utils';

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
      "flex items-center gap-2.5 px-3 py-2 rounded-md transition-colors text-sm",
      active 
        ? "bg-secondary text-secondary-foreground font-medium" 
        : "text-muted-foreground hover:bg-secondary/50 hover:text-foreground font-medium"
    )}
  >
    {icon}
    <span>{label}</span>
  </Link>
);

export const DashboardLayout = ({ children }: { children: React.ReactNode }) => {
  const location = useLocation();

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
          <div className="flex items-center gap-2 px-2 py-1.5 rounded bg-green-500/10 border border-green-500/20 text-green-700">
            <div className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse" />
            <span className="text-xs font-medium">Engine Connected</span>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 flex flex-col min-w-0">
        <header className="h-12 border-b border-border bg-card flex items-center justify-between px-4 shrink-0">
          <div className="flex items-center gap-2 text-sm">
             <span className="font-semibold text-foreground">
               {[...primaryItems, ...secondaryItems].find(item => location.pathname === item.to || (item.to !== '/' && location.pathname.startsWith(item.to)))?.label || 'Dashboard'}
             </span>
          </div>
          <div className="flex items-center gap-3">
             <div className="relative">
               <Search className="absolute left-2.5 top-1.5 text-muted-foreground" size={14} />
               <input 
                 type="text" 
                 placeholder="Search workflows, tasks..." 
                 className="h-8 w-64 bg-secondary/50 border border-border rounded-md pl-8 pr-3 text-xs focus:outline-none focus:ring-1 focus:ring-primary focus:bg-background transition-colors"
               />
               <div className="absolute right-2 top-1.5 text-[10px] text-muted-foreground border border-border rounded px-1 bg-background">⌘K</div>
             </div>
             <button className="relative p-1.5 text-muted-foreground hover:bg-secondary rounded-md transition-colors">
               <Bell size={16} />
               <span className="absolute top-1.5 right-1.5 w-1.5 h-1.5 bg-blue-500 rounded-full border border-card"></span>
             </button>
             <div className="w-7 h-7 rounded bg-primary/10 border border-primary/20 flex items-center justify-center text-primary font-semibold text-xs ml-2">
               Admin
             </div>
          </div>
        </header>
        
        <div className="flex-1 overflow-auto bg-muted/20">
          {children}
        </div>
      </main>
    </div>
  );
};
