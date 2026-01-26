import { useQuery } from '@tanstack/react-query';
import { api, Event } from '../../../lib/api';

interface UseHistoryEventsParams {
  from?: Date;
  to?: Date;
  limit?: number;
}

export function useHistoryEvents({ from, to, limit = 100 }: UseHistoryEventsParams) {
  return useQuery<Event[], Error>({
    queryKey: ['events', { from: from?.toISOString(), to: to?.toISOString(), limit }],
    queryFn: () => api.fetchEvents({
      from: from?.toISOString(),
      to: to?.toISOString(),
      limit
    }),
    refetchInterval: 5000, // Live-ish updates
    keepPreviousData: true,
  });
}
