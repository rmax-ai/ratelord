import { useState, useEffect } from 'react';
import { Clock, Calendar } from 'lucide-react';
import { cn } from '../../../lib/utils';

interface TimeRangePickerProps {
  from: Date | undefined;
  to: Date | undefined;
  onChange: (from: Date, to: Date) => void;
  className?: string;
}

const PRESETS = [
  { label: '1h', minutes: 60 },
  { label: '6h', minutes: 60 * 6 },
  { label: '24h', minutes: 60 * 24 },
  { label: '7d', minutes: 60 * 24 * 7 },
];

export function TimeRangePicker({ from, to, onChange, className }: TimeRangePickerProps) {
  // Local state for inputs to allow typing before committing
  const [localFrom, setLocalFrom] = useState(from ? from.toISOString().slice(0, 16) : '');
  const [localTo, setLocalTo] = useState(to ? to.toISOString().slice(0, 16) : '');

  useEffect(() => {
    if (from) setLocalFrom(from.toISOString().slice(0, 16));
    if (to) setLocalTo(to.toISOString().slice(0, 16));
  }, [from, to]);

  const applyPreset = (minutes: number) => {
    const now = new Date();
    const start = new Date(now.getTime() - minutes * 60 * 1000);
    onChange(start, now);
  };

  const handleManualChange = () => {
    const newFrom = new Date(localFrom);
    const newTo = new Date(localTo);
    if (!isNaN(newFrom.getTime()) && !isNaN(newTo.getTime())) {
      onChange(newFrom, newTo);
    }
  };

  return (
    <div className={cn("flex flex-wrap items-center gap-4 bg-slate-900 p-2 rounded-lg border border-slate-800", className)}>
      <div className="flex items-center gap-1 text-slate-400">
        <Clock className="w-4 h-4" />
        <span className="text-sm font-medium">Range</span>
      </div>

      <div className="flex items-center gap-1 bg-slate-800 rounded-md p-0.5">
        {PRESETS.map((preset) => (
          <button
            key={preset.label}
            onClick={() => applyPreset(preset.minutes)}
            className="px-3 py-1 text-xs font-medium rounded hover:bg-slate-700 text-slate-300 transition-colors"
          >
            {preset.label}
          </button>
        ))}
      </div>

      <div className="h-4 w-px bg-slate-800" />

      <div className="flex items-center gap-2">
        <div className="relative">
          <Calendar className="w-3 h-3 absolute left-2 top-1/2 -translate-y-1/2 text-slate-500" />
          <input
            type="datetime-local"
            value={localFrom}
            onChange={(e) => setLocalFrom(e.target.value)}
            onBlur={handleManualChange}
            className="pl-7 pr-2 py-1 bg-slate-950 border border-slate-800 rounded text-xs text-slate-300 focus:outline-none focus:border-indigo-500"
          />
        </div>
        <span className="text-slate-500 text-xs">to</span>
        <div className="relative">
          <Calendar className="w-3 h-3 absolute left-2 top-1/2 -translate-y-1/2 text-slate-500" />
          <input
            type="datetime-local"
            value={localTo}
            onChange={(e) => setLocalTo(e.target.value)}
            onBlur={handleManualChange}
            className="pl-7 pr-2 py-1 bg-slate-950 border border-slate-800 rounded text-xs text-slate-300 focus:outline-none focus:border-indigo-500"
          />
        </div>
      </div>
    </div>
  );
}
