import time
from typing import Dict, Any

class TelemetryCollector:
    """Aggregates local latency and invocation counts."""
    def __init__(self, service_name: str):
        self.service_name = service_name
        self.records = []

    def record_event(self, event_name: str, duration_ms: float) -> bool:
        self.records.append({"event": event_name, "duration": duration_ms, "time": time.time()})
        return True

def run_telemetry() -> Dict[str, Any]:
    collector = TelemetryCollector("payment-service")
    collector.record_event("checkout_duration", 14.5)
    return {"status": "recorded", "count": len(collector.records)}
