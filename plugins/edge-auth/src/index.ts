import type { Mission } from "@kestrel/contracts";

export interface Env {
  /** Comma-separated SHA-256 fingerprints of revoked device certificates. */
  REVOKED_FINGERPRINTS: string;
}

type CfRequest = Request & { cf?: IncomingRequestCfProperties };

/**
 * Rejects requests that do not present a valid hospital-device client cert.
 * Cloudflare terminates mTLS and exposes the verification result on request.cf.
 */
export default {
  async fetch(request: CfRequest, env: Env): Promise<Response> {
    const cert = request.cf?.tlsClientAuth;
    if (!cert || cert.certVerified !== "SUCCESS") {
      return new Response("client certificate required", { status: 403 });
    }
    const revoked = new Set(env.REVOKED_FINGERPRINTS.split(",").filter(Boolean));
    if (revoked.has(cert.certFingerprintSHA256)) {
      return new Response("client certificate revoked", { status: 403 });
    }

    const headers = new Headers(request.headers);
    headers.set("X-Device-Cert-Subject", cert.certSubjectDN);
    headers.set("X-Device-Cert-Fingerprint", cert.certFingerprintSHA256);
    return fetch(new Request(request, { headers }));
  },
};

/** Narrow helper used by tests to check a mission payload carries a hospital id. */
export function hasOrigin(m: Pick<Mission, "origin">): boolean {
  return Boolean(m.origin);
}
