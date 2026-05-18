import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Plus, Play, MoreVertical, Edit2, Search, Filter, GitBranch, Clock } from 'lucide-react';
import { useWorkflowStore } from '@/store/useWorkflowStore';
import { cn } from '@/lib/utils';

export default function WorkflowsPage() {
  const { workflows, fetchWorkflows, isLoading } = useWorkflowStore();
  const navigate = useNavigate();
  const [search, setSearch] = useState('');

  useEffect(() => {
    fetchWorkflows();
  }, [fetchWorkflows]);

  const getStatusBadge = (status: string) => {
    switch (status.toUpperCase()) {
      case 'COMPLETED': return 'bg-green-500/10 text-green-600 border-green-500/20';
      case 'RUNNING': return 'bg-blue-500/10 text-blue-600 border-blue-500/20';
      case 'FAILED': return 'bg-red-500/10 text-red-600 border-red-500/20';
      default: return 'bg-gray-500/10 text-gray-600 border-gray-500/20';
    }
  };

  const filteredWorkflows = workflows.filter(wf => 
    wf.name.toLowerCase().includes(search.toLowerCase()) || 
    (wf.description && wf.description.toLowerCase().includes(search.toLowerCase()))
  );

  return (
    <div className="p-6 max-w-[1600px] mx-auto space-y-4">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-xl font-bold text-foreground">Workflows</h1>
          <p className="text-sm text-muted-foreground">Manage, monitor, and configure execution pipelines.</p>
        </div>
        <div className="flex items-center gap-2">
           <button className="flex items-center gap-2 bg-secondary text-secondary-foreground border border-border px-3 py-1.5 rounded-md font-medium text-xs hover:bg-secondary/80 transition-colors">
             <Filter size={14} />
             Filter
           </button>
           <button 
             onClick={() => navigate('/workflows/new')}
             className="flex items-center gap-2 bg-primary text-primary-foreground px-3 py-1.5 rounded-md font-medium text-xs hover:bg-primary/90 transition-colors"
           >
             <Plus size={14} />
             New Workflow
           </button>
        </div>
      </div>

      <div className="bg-card border border-border rounded-lg shadow-sm overflow-hidden flex flex-col">
        <div className="p-3 border-b border-border flex items-center gap-3">
          <div className="relative flex-1 max-w-md">
             <Search className="absolute left-2.5 top-2 text-muted-foreground" size={14} />
             <input 
               type="text" 
               placeholder="Search workflows by name or ID..." 
               value={search}
               onChange={(e) => setSearch(e.target.value)}
               className="h-8 w-full bg-background border border-border rounded-md pl-8 pr-3 text-xs focus:outline-none focus:ring-1 focus:ring-primary transition-colors"
             />
          </div>
        </div>

        {isLoading ? (
          <div className="flex justify-center items-center py-20 text-muted-foreground">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mr-3"></div>
            <span className="text-sm font-medium">Loading workflows...</span>
          </div>
        ) : filteredWorkflows.length === 0 ? (
          <div className="text-center py-20 bg-background/50">
            <GitBranch size={32} className="mx-auto text-muted-foreground mb-3 opacity-20" />
            <h3 className="text-sm font-semibold text-foreground mb-1">No workflows found</h3>
            <p className="text-xs text-muted-foreground mb-4">You don't have any workflows matching this criteria.</p>
            <button 
              onClick={() => navigate('/workflows/new')}
              className="text-xs font-medium text-primary hover:underline"
            >
              Create your first workflow &rarr;
            </button>
          </div>
        ) : (
          <div className="overflow-x-auto">
             <table className="w-full text-left border-collapse">
               <thead>
                 <tr className="bg-secondary/50 border-b border-border text-xs text-muted-foreground uppercase tracking-wider">
                   <th className="px-4 py-2 font-semibold">Workflow Name</th>
                   <th className="px-4 py-2 font-semibold">Status</th>
                   <th className="px-4 py-2 font-semibold">Created</th>
                   <th className="px-4 py-2 font-semibold">Tasks</th>
                   <th className="px-4 py-2 font-semibold text-right">Actions</th>
                 </tr>
               </thead>
               <tbody className="divide-y divide-border text-sm">
                 {filteredWorkflows.map((workflow) => (
                   <tr key={workflow.id} className="hover:bg-secondary/20 transition-colors group">
                     <td className="px-4 py-3">
                       <div className="flex flex-col">
                         <span className="font-semibold text-foreground">{workflow.name}</span>
                         <span className="text-xs text-muted-foreground truncate max-w-[300px]">
                           {workflow.description || workflow.id}
                         </span>
                       </div>
                     </td>
                     <td className="px-4 py-3">
                       <span className={cn("px-2 py-0.5 rounded text-[10px] font-bold border uppercase", getStatusBadge(workflow.status || 'UNKNOWN'))}>
                         {workflow.status || 'UNKNOWN'}
                       </span>
                     </td>
                     <td className="px-4 py-3 text-muted-foreground text-xs">
                       <div className="flex items-center gap-1.5">
                         <Clock size={12} />
                         {new Date(workflow.createdAt).toLocaleString()}
                       </div>
                     </td>
                     <td className="px-4 py-3 text-muted-foreground text-xs">
                       <div className="flex items-center gap-1.5">
                         <GitBranch size={12} />
                         {Object.keys(workflow.tasks || {}).length} nodes
                       </div>
                     </td>
                     <td className="px-4 py-3 text-right">
                       <div className="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                         <button 
                           onClick={() => navigate(`/workflows/${workflow.id}`)}
                           className="p-1.5 text-muted-foreground hover:text-primary hover:bg-secondary rounded transition-colors"
                           title="Edit Workflow"
                         >
                           <Edit2 size={14} />
                         </button>
                         <button 
                           className="p-1.5 text-muted-foreground hover:text-green-600 hover:bg-green-50 rounded transition-colors"
                           title="Run Workflow"
                         >
                           <Play size={14} />
                         </button>
                         <button 
                           className="p-1.5 text-muted-foreground hover:text-foreground hover:bg-secondary rounded transition-colors"
                         >
                           <MoreVertical size={14} />
                         </button>
                       </div>
                     </td>
                   </tr>
                 ))}
               </tbody>
             </table>
          </div>
        )}
      </div>
    </div>
  );
}
