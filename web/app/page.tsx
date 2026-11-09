import LinkDashboard from "@/components/link-dashboard";

export default function Home() {
  return <main className="shell"><header className="topbar"><a className="brand" href="/" aria-label="Linklet home"><span className="brand-mark">↗</span> linklet</a><span className="topbar-note">Link management</span><div className="avatar">L</div></header><LinkDashboard /></main>;
}
