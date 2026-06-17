#!/usr/bin/env bash
set -euo pipefail

APP_PATH="${1:-build/bin/starxo.app}"
BUNDLE_ID="${STARXO_BUNDLE_ID:-com.starxo.app}"
IDENTITY="${MACOS_CODESIGN_IDENTITY:--}"

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "error: macOS app signing must run on Darwin" >&2
  exit 1
fi

if [[ ! -d "${APP_PATH}" ]]; then
  echo "error: app bundle not found: ${APP_PATH}" >&2
  exit 1
fi

INFO_PLIST="${APP_PATH}/Contents/Info.plist"
if [[ ! -f "${INFO_PLIST}" ]]; then
  echo "error: Info.plist not found: ${INFO_PLIST}" >&2
  exit 1
fi

actual_bundle_id="$(/usr/libexec/PlistBuddy -c "Print :CFBundleIdentifier" "${INFO_PLIST}")"
if [[ "${actual_bundle_id}" != "${BUNDLE_ID}" ]]; then
  echo "error: expected bundle id ${BUNDLE_ID}, got ${actual_bundle_id}" >&2
  exit 1
fi
local_network_usage="$(/usr/libexec/PlistBuddy -c "Print :NSLocalNetworkUsageDescription" "${INFO_PLIST}" 2>/dev/null || true)"
if [[ -z "${local_network_usage}" ]]; then
  echo "error: NSLocalNetworkUsageDescription must be present and non-empty" >&2
  exit 1
fi
allows_local_networking="$(/usr/libexec/PlistBuddy -c "Print :NSAppTransportSecurity:NSAllowsLocalNetworking" "${INFO_PLIST}" 2>/dev/null || true)"
if [[ "${allows_local_networking}" != "true" ]]; then
  echo "error: NSAppTransportSecurity.NSAllowsLocalNetworking must be true" >&2
  exit 1
fi

codesign --force --deep --sign "${IDENTITY}" "${APP_PATH}"
codesign --verify --deep --strict "${APP_PATH}"

signing_details="$(codesign -dv "${APP_PATH}" 2>&1)"
printf '%s\n' "${signing_details}"

if ! grep -q "Identifier=${BUNDLE_ID}" <<<"${signing_details}"; then
  echo "error: signed bundle identifier is not ${BUNDLE_ID}" >&2
  exit 1
fi
if grep -q "Info.plist=not bound" <<<"${signing_details}"; then
  echo "error: Info.plist is not bound into the app signature" >&2
  exit 1
fi
if grep -q "Sealed Resources=none" <<<"${signing_details}"; then
  echo "error: app resources are not sealed" >&2
  exit 1
fi
