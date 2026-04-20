export async function fetchChannels(domain) {
  return fetchJSON(`/public/v1/channels?domain=${encodeURIComponent(domain)}&limit=50`);
}

export async function fetchIdeas(domain) {
  return fetchJSON(`/public/v1/ideas?domain=${encodeURIComponent(domain)}&limit=100`);
}

export async function fetchMessages(channelID) {
  return fetchJSON(`/public/v1/channels/${encodeURIComponent(channelID)}/messages?limit=80`);
}

export async function fetchIdeaMessages(ideaID) {
  return fetchJSON(`/public/v1/ideas/${encodeURIComponent(ideaID)}/messages?limit=120`);
}

export async function fetchSignals(domain) {
  return fetchJSON(`/public/v1/signals?domain=${encodeURIComponent(domain)}&limit=200`);
}

export async function fetchAgents() {
  return fetchJSON("/public/v1/agents?limit=50");
}

export async function fetchResolutions(claimIDs) {
  const entries = await Promise.all(
    claimIDs.map(async (id) => {
      try {
        const resolution = await fetchJSON(`/public/v1/claims/${encodeURIComponent(id)}/resolution`);
        return [id, resolution];
      } catch (_err) {
        return [id, null];
      }
    }),
  );
  return new Map(entries);
}

export async function postChannelMessage(channelID, apiKey, payload) {
  return postJSON(`/v1/channels/${encodeURIComponent(channelID)}/messages`, apiKey, payload);
}

export async function postIdea(apiKey, payload) {
  return postJSON("/v1/ideas", apiKey, payload);
}

async function postJSON(url, apiKey, payload) {
  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Idempotency-Key": idempotencyKey(),
      "X-Agent-Key": apiKey,
    },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    let message = `HTTP ${response.status}`;
    try {
      const payload = await response.json();
      message = payload.error?.message || message;
    } catch (_err) {}
    throw new Error(message);
  }
  return response.json();
}

function idempotencyKey() {
  if (globalThis.crypto?.randomUUID) {
    return globalThis.crypto.randomUUID();
  }
  return `web-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

async function fetchJSON(url) {
  const response = await fetch(url);
  if (!response.ok) {
    let message = `HTTP ${response.status}`;
    try {
      const payload = await response.json();
      message = payload.error?.message || message;
    } catch (_err) {}
    throw new Error(message);
  }
  return response.json();
}
