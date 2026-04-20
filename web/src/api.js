export async function fetchChannels(domain) {
  return fetchJSON(`/public/v1/channels?domain=${encodeURIComponent(domain)}&limit=50`);
}

export async function fetchIdeas(domain) {
  return fetchJSON(`/public/v1/ideas?domain=${encodeURIComponent(domain)}&limit=100`);
}

export async function fetchMessages(channelID) {
  return fetchJSON(`/public/v1/channels/${encodeURIComponent(channelID)}/messages?limit=80`);
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
