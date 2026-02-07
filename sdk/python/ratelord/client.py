import requests
import time
from typing import Optional
from tenacity import (
    retry,
    stop_after_attempt,
    wait_exponential,
    wait_random_exponential,
    retry_if_exception_type,
)
from .types import Intent, Decision, Mods


class Client:
    def __init__(self, endpoint: str = "http://127.0.0.1:8090"):
        self.endpoint = endpoint

    @retry(
        stop=stop_after_attempt(5),
        wait=wait_random_exponential(multiplier=0.1, max=1.0),
        retry=retry_if_exception_type(requests.RequestException),
        reraise=True,
    )
    def _post_intent(self, payload: dict) -> requests.Response:
        resp = requests.post(
            f"{self.endpoint}/v1/intent",
            json=payload,
            timeout=10,
        )
        # Raise exception for 5xx errors to trigger retry
        if resp.status_code >= 500:
            resp.raise_for_status()
        return resp

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
            resp = self._post_intent(
                {
                    "agent_id": intent.agent_id,
                    "identity_id": intent.identity_id,
                    "workload_id": intent.workload_id,
                    "scope_id": intent.scope_id,
                    "urgency": intent.urgency,
                    "expected_cost": intent.expected_cost,
                    "duration_hint": intent.duration_hint,
                    "client_context": intent.client_context,
                }
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
                    # In a real async client we wouldn't block, but this is sync
                    time.sleep(mods.wait_seconds)
                return Decision(
                    allowed=data["status"] in ["approve", "approve_with_modifications"],
                    intent_id=data["intent_id"],
                    status=data["status"],
                    modifications=mods,
                    reason=data.get("reason"),
                )
            else:
                # 4xx errors or others that didn't raise (shouldn't happen for 5xx due to _post_intent)
                return Decision(
                    allowed=False,
                    intent_id="",
                    status="deny_with_reason",
                    modifications=Mods(),
                    reason="upstream_error",
                )
        except requests.RequestException:
            # Catches both connection errors and the re-raised 5xx errors after retries exhausted
            return Decision(
                allowed=False,
                intent_id="",
                status="deny_with_reason",
                modifications=Mods(),
                reason="daemon_unreachable",
            )
