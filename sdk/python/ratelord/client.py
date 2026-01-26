import requests
import time
from typing import Optional
from .types import Intent, Decision, Mods


class Client:
    def __init__(self, endpoint: str = "http://127.0.0.1:8090"):
        self.endpoint = endpoint

    def ask(self, intent: Intent) -> Decision:
        # Validate mandatory fields
        if not all(
            [intent.agent_id, intent.identity_id, intent.workload_id, intent.scope_id]
        ):
            return Decision(
                allowed=False,
                intent_id="",
                status="deny_with_reason",
                modifications=Mods(),
                reason="invalid_intent",
            )

        try:
            resp = requests.post(
                f"{self.endpoint}/v1/intent",
                json={
                    "agent_id": intent.agent_id,
                    "identity_id": intent.identity_id,
                    "workload_id": intent.workload_id,
                    "scope_id": intent.scope_id,
                    "urgency": intent.urgency,
                    "expected_cost": intent.expected_cost,
                    "duration_hint": intent.duration_hint,
                    "client_context": intent.client_context,
                },
                timeout=10,
            )

            if resp.status_code == 200:
                data = resp.json()
                mods = Mods(
                    wait_seconds=data.get("modifications", {}).get("wait_seconds", 0.0),
                    identity_switch=data.get("modifications", {}).get(
                        "identity_switch"
                    ),
                )
                if mods.wait_seconds > 0:
                    print(f"Rate limiting: waiting {mods.wait_seconds} seconds...")
                    time.sleep(mods.wait_seconds)
                return Decision(
                    allowed=data["status"] in ["approve", "approve_with_modifications"],
                    intent_id=data["intent_id"],
                    status=data["status"],
                    modifications=mods,
                    reason=data.get("reason"),
                )
            else:
                return Decision(
                    allowed=False,
                    intent_id="",
                    status="deny_with_reason",
                    modifications=Mods(),
                    reason="upstream_error",
                )
        except requests.RequestException:
            return Decision(
                allowed=False,
                intent_id="",
                status="deny_with_reason",
                modifications=Mods(),
                reason="daemon_unreachable",
            )
