"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";

type Link = { id: number; code: string; url: string; created_at: string };
const api = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const publicBase = process.env.NEXT_PUBLIC_LINK_BASE_URL ?? api;

export default function LinkDashboard() {
  const [items, setItems] = useState<Link[]>([]);
  const [url, setUrl] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState("");

  const refresh = useCallback(async () => {
    try {
      const response = await fetch(`${api}/api/v1/links?limit=50`, { cache: "no-store" });
      if (response.ok) setItems((await response.json()).items as Link[]);
    } catch {
      setError("Could not reach the API. Check that the services are running.");
    }
  }, []);

  useEffect(() => { void refresh(); }, [refresh]);

  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true); setError("");
    try {
      const response = await fetch(`${api}/api/v1/links`, {
        method: "POST", headers: { "Content-Type": "application/json", "Idempotency-Key": crypto.randomUUID() },
        body: JSON.stringify({ url, ...(code ? { code } : {}) }),
      });
      const result = await response.json();
      if (!response.ok) throw new Error(result.detail ?? "Link could not be created.");
      setUrl(""); setCode(""); await refresh();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Link could not be created.");
    } finally { setBusy(false); }
  }

  async function copy(value: string) {
    await navigator.clipboard.writeText(value);
    setCopied(value);
    window.setTimeout(() => setCopied(""), 1600);
  }

  return <section className="content">
    <div className="intro"><div><p className="eyebrow">YOUR LINK WORKSPACE</p><h1>Make every link <span>count.</span></h1><p className="subtitle">Create clean, memorable links and keep your destinations organized.</p></div><div className="intro-icon" aria-hidden="true">↗</div></div>
    <form className="create-card" onSubmit={create}><div className="form-heading"><div><h2>Create a short link</h2><p>Paste a long URL to get started.</p></div><span className="secure">⌑ &nbsp; Secure links</span></div><label htmlFor="destination">Destination URL</label><div className="url-input"><span>↗</span><input id="destination" type="url" placeholder="https://example.com/your/long/link" value={url} onChange={e => setUrl(e.target.value)} required maxLength={2048} /></div><div className="form-bottom"><label className="custom-label" htmlFor="custom-code">Custom alias <span>Optional</span></label><div className="form-actions"><div className="alias-input"><span>lnk.lt/</span><input id="custom-code" value={code} onChange={e => setCode(e.target.value)} minLength={4} maxLength={32} pattern="[A-Za-z0-9_-]+" placeholder="my-link" /></div><button disabled={busy}>{busy ? "Creating…" : "Create link"}<span>→</span></button></div></div>{error && <p className="error" role="alert">{error}</p>}</form>
    <div className="list-heading"><div><p className="eyebrow">LINK LIBRARY</p><h2>Your links <span className="count">{items.length}</span></h2></div><button className="refresh" onClick={() => void refresh()} aria-label="Refresh links">↻</button></div>
    <div className="links-card">{items.length === 0 ? <div className="empty"><div>↗</div><h3>Your next great link starts here</h3><p>Short links you create will show up in this space.</p></div> : items.map(item => { const short = `${publicBase}/r/${item.code}`; return <article className="link-row" key={item.id}><div className="link-icon">↗</div><div className="link-info"><button className="short-url" onClick={() => void copy(short)}>{short}</button><p>{item.url}</p></div><time dateTime={item.created_at}>{new Date(item.created_at).toLocaleDateString()}</time><button className="copy-button" onClick={() => void copy(short)} aria-label="Copy short link">{copied === short ? "Copied" : "Copy"}</button></article>; })}</div>
    <footer><span>Built for links that go places.</span><span>LINKLET&nbsp; · &nbsp;API v1</span></footer>
  </section>;
}
