import { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { TimeRangePicker } from './components/TimeRangePicker';
import { EventTimeline } from './components/EventTimeline';
import { EventList } from './components/EventList';
import { EventDetailPanel } from './components/EventDetailPanel';
import { useHistoryEvents } from './hooks/useHistoryEvents';

export default function HistoryView() {
  const [searchParams, setSearchParams] = useSearchParams();
  
  // Parse URL params for time range
  const getInitialDate = (param: string, fallback: Date) => {
    const val = searchParams.get(param);
    if (!val) return fallback;
    const d = new Date(val);
    return isNaN(d.getTime()) ? fallback : d;
  };

  const now = new Date();
  const [dateRange, setDateRange] = useState({
    from: getInitialDate('from', new Date(now.getTime() - 60 * 60 * 1000)), // Default 1h
    to: getInitialDate('to', now)
  });

  const [selectedEventId, setSelectedEventId] = useState<string | null>(null);

  // Sync state to URL
  useEffect(() => {
    const params = new URLSearchParams(searchParams);
    params.set('from', dateRange.from.toISOString());
    params.set('to', dateRange.to.toISOString());
    setSearchParams(params, { replace: true });
  }, [dateRange, setSearchParams]);

  // Data fetching
  const { data: events = [], isLoading, error } = useHistoryEvents({
    from: dateRange.from,
    to: dateRange.to,
    limit: 500 // Cap for performance
  });

  const selectedEvent = events.find(e => e.event_id === selectedEventId) || null;

  const handleTimeChange = (from: Date, to: Date) => {
    setDateRange({ from, to });
  };

  return (
    <div className="h-full flex flex-col bg-slate-950">
      {/* Toolbar */}
      <div className="flex-none p-4 border-b border-slate-900 bg-slate-950/50 backdrop-blur-sm z-10">
        <div className="flex items-center justify-between mb-4">
          <h1 className="text-xl font-semibold text-slate-100">Event History</h1>
          <div className="text-xs text-slate-500">
            {events.length} events loaded
          </div>
        </div>
        <TimeRangePicker 
          from={dateRange.from} 
          to={dateRange.to} 
          onChange={handleTimeChange} 
          className="w-full"
        />
      </div>

      {/* Timeline Visualization */}
      <div className="flex-none p-4 border-b border-slate-900 bg-slate-950">
         <EventTimeline events={events} />
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-hidden relative flex">
        {/* Loading/Error States */}
        {isLoading && (
          <div className="absolute inset-0 z-20 bg-slate-950/80 flex items-center justify-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
          </div>
        )}
        
        {error ? (
          <div className="absolute inset-0 z-20 flex items-center justify-center">
             <div className="text-red-400 bg-red-900/20 p-4 rounded-lg border border-red-900">
               Error loading events: {error.message}
             </div>
          </div>
        ) : (
           <EventList 
             events={events} 
             onSelect={(e) => setSelectedEventId(e.event_id)} 
             selectedId={selectedEventId}
           />
        )}

        {/* Detail Panel */}
        {selectedEvent && (
          <EventDetailPanel 
            event={selectedEvent} 
            onClose={() => setSelectedEventId(null)} 
          />
        )}
      </div>
    </div>
  );
}
