import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import { Event } from '../../../lib/api';
import { useMemo } from 'react';

interface EventTimelineProps {
  events: Event[];
  className?: string;
}

export function EventTimeline({ events, className }: EventTimelineProps) {
  const chartData = useMemo(() => {
    if (!events.length) return [];

    // Bucket events by hour or minute based on range
    const sorted = [...events].sort((a, b) => new Date(a.ts_event).getTime() - new Date(b.ts_event).getTime());
    const start = new Date(sorted[0].ts_event).getTime();
    const end = new Date(sorted[sorted.length - 1].ts_event).getTime();
    const diff = end - start;
    
    // Determine bucket size (approx 40 buckets)
    const bucketSize = Math.max(diff / 40, 60 * 1000); // Min 1 minute

    const buckets = new Map<number, number>();
    
    sorted.forEach(evt => {
      const ts = new Date(evt.ts_event).getTime();
      const bucket = Math.floor(ts / bucketSize) * bucketSize;
      buckets.set(bucket, (buckets.get(bucket) || 0) + 1);
    });

    return Array.from(buckets.entries())
      .map(([ts, count]) => ({
        timestamp: ts,
        timeLabel: new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        count
      }))
      .sort((a, b) => a.timestamp - b.timestamp);

  }, [events]);

  if (!events.length) {
    return (
      <div className={`h-32 flex items-center justify-center bg-slate-900/50 rounded-lg border border-slate-800 border-dashed text-slate-500 text-sm ${className}`}>
        No events in range
      </div>
    );
  }

  return (
    <div className={`h-32 w-full ${className}`}>
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={chartData}>
          <XAxis 
            dataKey="timeLabel" 
            stroke="#475569" 
            fontSize={10} 
            tickLine={false}
            axisLine={false}
            minTickGap={30}
          />
          <Tooltip 
            contentStyle={{ backgroundColor: '#0f172a', borderColor: '#1e293b', color: '#f8fafc' }}
            cursor={{ fill: '#1e293b' }}
          />
          <Bar dataKey="count" fill="#6366f1" radius={[2, 2, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
