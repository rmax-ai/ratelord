import { RatelordClient } from './client';
import { RatelordIntent } from './types';

// Mock global fetch
const mockFetch = jest.fn();
global.fetch = mockFetch;

describe('RatelordClient', () => {
  let client: RatelordClient;

  beforeEach(() => {
    mockFetch.mockClear();
    client = new RatelordClient({ endpoint: 'http://test-api' });
  });

  it('should send a valid intent and return decision', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        allowed: true,
        intent_id: 'test-intent-id',
        status: 'approve',
      }),
    });

    const intent: RatelordIntent = {
      agentId: 'agent-1',
      identityId: 'id-1',
      workloadId: 'work-1',
      scopeId: 'scope-1',
    };

    const decision = await client.ask(intent);

    expect(mockFetch).toHaveBeenCalledWith(
      'http://test-api/v1/intent',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          agent_id: 'agent-1',
          identity_id: 'id-1',
          workload_id: 'work-1',
          scope_id: 'scope-1',
        }),
      })
    );

    expect(decision).toEqual({
      allowed: true,
      intentId: 'test-intent-id',
      status: 'approve',
      reason: undefined,
      modifications: undefined,
    });
  });

  it('should fail-closed if daemon returns 500', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      statusText: 'Internal Server Error',
    });

    const intent: RatelordIntent = {
      agentId: 'agent-1',
      identityId: 'id-1',
      workloadId: 'work-1',
      scopeId: 'scope-1',
    };

    const decision = await client.ask(intent);

    expect(decision.allowed).toBe(false);
    expect(decision.status).toBe('deny_with_reason');
    expect(decision.reason).toContain('upstream_error');
  });

  it('should fail-closed if network throws', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    const intent: RatelordIntent = {
      agentId: 'agent-1',
      identityId: 'id-1',
      workloadId: 'work-1',
      scopeId: 'scope-1',
    };

    const decision = await client.ask(intent);

    expect(decision.allowed).toBe(false);
    expect(decision.status).toBe('deny_with_reason');
    expect(decision.reason).toContain('daemon_unreachable');
  });

  it('should sleep if waitSeconds is provided', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        allowed: true,
        intent_id: 'delayed-intent',
        status: 'approve_with_modifications',
        modifications: {
          wait_seconds: 0.1, // 100ms
        },
      }),
    });

    const start = Date.now();
    const intent: RatelordIntent = {
      agentId: 'agent-1',
      identityId: 'id-1',
      workloadId: 'work-1',
      scopeId: 'scope-1',
    };

    const decision = await client.ask(intent);
    const duration = Date.now() - start;

    expect(decision.allowed).toBe(true);
    // Ensure it waited at least 100ms (allow some buffer for test execution)
    expect(duration).toBeGreaterThanOrEqual(90); 
    expect(decision.modifications?.waitSeconds).toBe(0.1);
  });
});
