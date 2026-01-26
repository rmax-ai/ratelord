import { X, Copy, Check } from 'lucide-react';
import { cn } from '../../../lib/utils';
import { Event } from '../../../lib/api';
import { useState } from 'react';

interface EventDetailPanelProps {
  event: Event | null;
  onClose: () => void;
}

export function EventDetailPanel({ event, onClose }: EventDetailPanelProps) {
  const [copied, setCopied] = useState(false);

  if (!event) return null;

  const handleCopy = () => {
    navigator.clipboard.writeText(JSON.stringify(event, null, 2));
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="fixed inset-y-0 right-0 w-[600px] bg-slate-900 border-l border-slate-800 shadow-2xl transform transition-transform duration-300 ease-in-out z-50 flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800 bg-slate-900/50 backdrop-blur-sm">
        <div>
          <h2 className="text-lg font-semibold text-slate-100">Event Details</h2>
          <p className="text-xs text-slate-400 font-mono">{event.event_id}</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleCopy}
            className="p-2 text-slate-400 hover:text-indigo-400 transition-colors rounded-md hover:bg-slate-800"
            title="Copy JSON"
          >
            {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
          </button>
          <button
            onClick={onClose}
            className="p-2 text-slate-400 hover:text-red-400 transition-colors rounded-md hover:bg-slate-800"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {/* Metadata Grid */}
        <div className="grid grid-cols-2 gap-4">
          <div className="bg-slate-950 p-3 rounded-lg border border-slate-800">
            <span className="text-xs text-slate-500 uppercase tracking-wider font-semibold">Type</span>
            <div className="mt-1 text-sm text-indigo-400 font-mono">{event.event_type}</div>
          </div>
          <div className="bg-slate-950 p-3 rounded-lg border border-slate-800">
            <span className="text-xs text-slate-500 uppercase tracking-wider font-semibold">Timestamp</span>
            <div className="mt-1 text-sm text-slate-300 font-mono">
              {new Date(event.ts_event).toLocaleString()}
            </div>
          </div>
          <div className="bg-slate-950 p-3 rounded-lg border border-slate-800 col-span-2">
            <span className="text-xs text-slate-500 uppercase tracking-wider font-semibold">Dimensions</span>
            <div className="mt-2 grid grid-cols-2 gap-2 text-xs font-mono text-slate-400">
              <div className="flex justify-between">
                <span>Agent:</span>
                <span className="text-slate-300">{event.dimensions.agent_id}</span>
              </div>
              <div className="flex justify-between">
                <span>Scope:</span>
                <span className="text-slate-300">{event.dimensions.scope_id}</span>
              </div>
              <div className="flex justify-between">
                <span>Identity:</span>
                <span className="text-slate-300">{event.dimensions.identity_id}</span>
              </div>
              <div className="flex justify-between">
                <span>Workload:</span>
                <span className="text-slate-300">{event.dimensions.workload_id}</span>
              </div>
            </div>
          </div>
        </div>

        {/* JSON Payload */}
        <div>
          <span className="text-xs text-slate-500 uppercase tracking-wider font-semibold mb-2 block">Payload</span>
          <div className="bg-slate-950 rounded-lg border border-slate-800 p-4 overflow-x-auto">
            <pre className="text-xs text-green-400 font-mono leading-relaxed">
              {JSON.stringify(event.payload, null, 2)}
            </pre>
          </div>
        </div>
        
        {/* Source Info */}
        <div>
           <span className="text-xs text-slate-500 uppercase tracking-wider font-semibold mb-2 block">Source Metadata</span>
           <div className="bg-slate-950 rounded-lg border border-slate-800 p-4 text-xs font-mono text-slate-400 space-y-1">
             <div className="flex gap-2">
                <span className="text-slate-600">Origin:</span>
                <span>{event.source.origin_kind}::{event.source.origin_id}</span>
             </div>
             <div className="flex gap-2">
                <span className="text-slate-600">Writer:</span>
                <span>{event.source.writer_id}</span>
             </div>
              <div className="flex gap-2">
                <span className="text-slate-600">Correlation:</span>
                <span>{event.correlation.correlation_id}</span>
             </div>
           </div>
        </div>
      </div>
    </div>
  );
}
