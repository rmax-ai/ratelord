import { RatelordClient } from '../src/client';

async function main() {
  // Instantiate the client with default options
  const client = new RatelordClient();

  // Sample intent
  const intent = {
    agentId: "example-agent",
    identityId: "dev-user",
    workloadId: "test-run",
    scopeId: "api-read"
  };

  try {
    // Call ask() to negotiate the intent
    const decision = await client.ask(intent);

    // Log the decision
    console.log('Decision:', decision);

    if (decision.allowed) {
      console.log('Intent approved! Proceeding with the action.');
    } else {
      console.log('Intent denied:', decision.reason);
    }
  } catch (error) {
    // Handle any unexpected errors
    console.error('Unexpected error:', error);
  }
}

// Run the example
main();