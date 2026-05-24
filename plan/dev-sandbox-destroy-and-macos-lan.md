# Dev Sandbox Destroy and macOS LAN Fix

## Summary
- Fix two dev integration regressions:
  - Destroying the active sandbox must not disconnect SSH.
  - macOS app bundles must use a Starxo-owned bundle identity and be signed before release packaging.
- Implement as two commits:
  - `fix(sandbox): keep ssh connected when destroying active sandbox`
  - `build(mac): sign app bundle with starxo identity`

## Sandbox Lifecycle
- `ContainerService.DestroyContainer` must treat sandbox deletion and SSH disconnection as separate lifecycle actions.
- Deleting the active sandbox removes its remote workspace, clears only the active sandbox, emits sandbox deactivation/destruction events, and keeps the SSH manager plus health monitor alive.
- Remote deletion failure leaves the active sandbox state intact so the user can retry.

## macOS Bundle Identity
- macOS bundle identifier is `com.starxo.app`.
- Release builds sign the `.app` bundle after Wails build and before packaging.
- Signing may be ad-hoc for v1, but the signed bundle must bind `Info.plist` and seal resources so macOS Local Network permission is associated with the Starxo bundle identity.

## Verification
- Run Go tests, frontend build, Wails build, bundle signing checks, and manual SSH/sandbox regression against `192.168.31.59`.
