#!/usr/bin/env python3

from ratelord import Client, Intent


def main():
    # Create a client with default endpoint
    client = Client()

    # Create an intent to ask for permission
    intent = Intent(
        agent_id="example-agent",
        identity_id="pat:example-token",
        workload_id="example_workload",
        scope_id="repo:example/repo",
    )

    # Ask for permission
    decision = client.ask(intent)

    # Print the result
    if decision.allowed:
        print(f"Approved: {decision.status}")
        if decision.intent_id:
            print(f"Intent ID: {decision.intent_id}")
    else:
        print(f"Denied: {decision.reason}")


if __name__ == "__main__":
    main()
