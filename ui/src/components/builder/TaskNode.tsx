import { memo } from 'react';
import { Handle, Position, type Node, type NodeProps } from '@xyflow/react';
import { CheckCircle2, Clock, AlertCircle, PlayCircle, MoreHorizontal } from 'lucide-react';
import { cn } from '@/lib/utils';

export type TaskNodeData = {
  label: string;
  status: string;
  description?: string;
  provider?: string;
  model?: string;
  onEdit?: () => void;
};

const TaskNode = ({ data, selected }: NodeProps<Node<TaskNodeData>>) => {
  const getStatusIcon = () => {
    switch (data.status?.toUpperCase()) {
      case 'COMPLETED': return <CheckCircle2 className="text-green-500" size={14} />;
      case 'RUNNING': return <PlayCircle className="text-blue-500 animate-pulse" size={14} />;
      case 'FAILED': return <AlertCircle className="text-red-500" size={14} />;
      case 'WAITING_FOR_APPROVAL': return <Clock className="text-amber-500" size={14} />;
      default: return <Clock className="text-muted-foreground" size={14} />;
    }
  };

  const getStatusColor = () => {
    switch (data.status?.toUpperCase()) {
      case 'COMPLETED': return 'bg-green-500';
      case 'RUNNING': return 'bg-blue-500';
      case 'FAILED': return 'bg-red-500';
      case 'WAITING_FOR_APPROVAL': return 'bg-amber-500';
      default: return 'bg-gray-400';
    }
  };

  return (
    <div className={cn(
      "w-[260px] shadow-sm rounded-md bg-card border transition-all flex flex-col overflow-hidden text-left",
      selected ? "border-primary ring-1 ring-primary" : "border-border hover:border-muted-foreground/50"
    )}>
      {/* Top Status Bar */}
      <div className={cn("h-1 w-full", getStatusColor())} />
      
      <Handle type="target" position={Position.Top} className="w-2.5 h-2.5 bg-muted-foreground border-card -mt-1.5" />
      
      <div className="p-3">
        <div className="flex items-start justify-between mb-1">
          <div className="flex items-center gap-2">
            {getStatusIcon()}
            <span className="font-semibold text-sm text-foreground truncate">{data.label}</span>
          </div>
          <button 
            onClick={(e) => {
              e.stopPropagation();
              data.onEdit?.();
            }}
            className="text-muted-foreground hover:text-foreground transition-colors p-0.5 rounded-sm hover:bg-secondary"
          >
            <MoreHorizontal size={14} />
          </button>
        </div>

        {data.description && (
          <div className="text-xs text-muted-foreground line-clamp-2 mt-1 leading-snug">
            {data.description}
          </div>
        )}

        {/* Tags / Config indicators */}
        <div className="flex flex-wrap items-center gap-1.5 mt-3 pt-2 border-t border-border/50">
          {data.provider && (
            <span className="px-1.5 py-0.5 bg-secondary text-secondary-foreground text-[10px] font-medium rounded uppercase tracking-wider">
              {data.provider}
            </span>
          )}
          {data.model && (
            <span className="px-1.5 py-0.5 border border-border text-muted-foreground text-[10px] font-medium rounded truncate max-w-[120px]">
              {data.model}
            </span>
          )}
          {!data.provider && !data.model && (
            <span className="text-[10px] text-muted-foreground/50 italic">Unconfigured</span>
          )}
        </div>
      </div>

      <Handle type="source" position={Position.Bottom} className="w-2.5 h-2.5 bg-muted-foreground border-card -mb-1.5" />
    </div>
  );
};

export default memo(TaskNode);
