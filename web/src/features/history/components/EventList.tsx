import { Event } from '../../../lib/api';
import { cn } from '../../../lib/utils';
import { ChevronRight } from 'lucide-react';

interface EventListProps {
  events: Event[];
  onSelect: (event: Event) => void;
  selectedId: string | null;
}

export function EventList({ events, onSelect, selectedId }: EventListProps) {
  if (!events.length) {
    return (
      <div className="flex-1 flex items-center justify-center text-slate-500">
        No events found
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-auto">
      <table className="w-full text-left border-collapse">
        <thead className="bg-slate-900 sticky top-0 z-10 shadow-sm">
          <tr>
            <th className="p-3 text-xs font-medium text-slate-500 uppercase tracking-wider border-b border-slate-800">Time</th>
            <th className="p-3 text-xs font-medium text-slate-500 uppercase tracking-wider border-b border-slate-800">Type</th>
            <th className="p-3 text-xs font-medium text-slate-500 uppercase tracking-wider border-b border-slate-800">Agent</th>
            <th className="p-3 text-xs font-medium text-slate-500 uppercase tracking-wider border-b border-slate-800">Decision</th>
            <th className="p-3 w-8 border-b border-slate-800"></th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800/50">
          {events.map((event) => {
            const isSelected = event.event_id === selectedId;
            return (
              <tr 
                key={event.event_id}
                onClick={() => onSelect(event)}
                className={cn(
                  "cursor-pointer transition-colors hover:bg-slate-800/50",
                  isSelected ? "bg-indigo-900/20 hover:bg-indigo-900/30" : "bg-transparent"
                )}
              >
                <td className="p-3 text-sm text-slate-300 whitespace-nowrap font-mono text-xs">
                  {new Date(event.ts_event).toLocaleString()}
                </td>
                <td className="p-3">
                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-800 text-indigo-400 border border-slate-700 font-mono">
                    {event.event_type}
                  </span>
                </td>
                <td className="p-3 text-sm text-slate-400 font-mono text-xs">
                  {event.dimensions.agent_id}
                </td>
                <td className="p-3 text-sm text-slate-300">
                  {/* Safely extract decision or relevant payload info */}
                  {event.payload?.decision ? (
                    <span className={cn(
                      "inline-flex items-center px-2 py-0.5 rounded text-xs font-medium border font-mono lowercase",
                      event.payload.decision === 'allow' ? "bg-green-900/30 text-green-400 border-green-900" :
                      event.payload.decision === 'deny' ? "bg-red-900/30 text-red-400 border-red-900" :
                      "bg-slate-800 text-slate-300 border-slate-700"
                    )}>
                      {event.payload.decision}
                    </span>
                  ) : (
                    <span className="text-slate-600 text-xs">-</span>
                  )}
                </td>
                <td className="p-3 text-slate-600">
                  <ChevronRight className={cn("w-4 h-4 transition-transform", isSelected && "text-indigo-400 translate-x-1")} />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
