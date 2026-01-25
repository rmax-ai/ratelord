# NEXT STEPS: Phase 4 - Implementation & Verification

The provider integration is now complete, with polling orchestrator implemented. The next step is to implement the forecasting engine to predict time-to-exhaustion.

## Current Objective: Epic 7 - Forecasting (Prediction Engine)

We need to implement the forecast models and loop to compute predictions based on usage history.

### Tasks for Next Session:
1. **Implement Forecast Model Interface**:
   - Create `pkg/engine/forecast/types.go`.
   - Define `Model` interface (Inputs -> Distribution).
   - M7.1: Forecast Model Interface.

2. **Implement Linear Burn Model**:
   - Implement simple linear regression model.
   - Calculate P99 time-to-exhaustion based on recent history.
   - M7.2: Linear Burn Model.

3. **Implement Forecast Loop**:
   - Trigger forecasts after `usage_observed` events.
   - Emit `forecast_computed` events.
   - M7.3: Forecast Loop.

## Reference
- **Plan**: `TASKS.md` (Epic 7)
- **Architecture**: `ARCHITECTURE.md` (Prediction Engine)
