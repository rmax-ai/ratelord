from dataclasses import dataclass
from typing import Dict, Any, Optional


@dataclass
class Intent:
    agent_id: str
    identity_id: str
    workload_id: str
    scope_id: str
    urgency: str = "normal"
    expected_cost: float = 1.0
    duration_hint: Optional[float] = None
    client_context: Optional[Dict[str, Any]] = None


@dataclass
class Mods:
    wait_seconds: float = 0.0
    identity_switch: Optional[str] = None


@dataclass
class Decision:
    allowed: bool
    intent_id: str
    status: str
    modifications: Mods
    reason: Optional[str] = None
