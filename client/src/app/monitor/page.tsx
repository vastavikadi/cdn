"use client";

import { useCallback, useEffect, useState } from "react";

const EDGES = [
  {
    id: "india",
    name: "INDIA",
    url: "http://localhost:8081",
  },
  {
    id: "europe",
    name: "EUROPE",
    url: "http://localhost:8082",
  },
];

interface EdgeStats {
  edge_id: string;
  region: string;
  cache_hits: number;
  cache_misses: number;
  cached_assets: number;
  cache_size_bytes: number;
}

interface EdgeResult {
  id: string;
  name: string;
  url: string;
  stats: EdgeStats | null;
  online: boolean;
}

export default function MonitorPage() {
  const [edges, setEdges] = useState<EdgeResult[]>(
    EDGES.map((edge) => ({
      ...edge,
      stats: null,
      online: false,
    })),
  );

  const [lastUpdated, setLastUpdated] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchStats = useCallback(async () => {
    const results = await Promise.all(
      EDGES.map(async (edge): Promise<EdgeResult> => {
        try {
          const response = await fetch(`${edge.url}/stats`, {
            cache: "no-store",
            signal: AbortSignal.timeout(4000),
          });

          if (!response.ok) {
            throw new Error("Stats endpoint unavailable");
          }

          const stats: EdgeStats = await response.json();

          return {
            ...edge,
            stats,
            online: true,
          };
        } catch {
          return {
            ...edge,
            stats: null,
            online: false,
          };
        }
      }),
    );

    setEdges(results);
    setLastUpdated(new Date().toLocaleTimeString());
    setLoading(false);
  }, []);

  useEffect(() => {
    fetchStats();

    const interval = setInterval(fetchStats, 5000);

    return () => clearInterval(interval);
  }, [fetchStats]);

  const totalHits = edges.reduce(
    (sum, edge) => sum + (edge.stats?.cache_hits ?? 0),
    0,
  );

  const totalMisses = edges.reduce(
    (sum, edge) => sum + (edge.stats?.cache_misses ?? 0),
    0,
  );

  const totalAssets = edges.reduce(
    (sum, edge) => sum + (edge.stats?.cached_assets ?? 0),
    0,
  );

  const totalBytes = edges.reduce(
    (sum, edge) => sum + (edge.stats?.cache_size_bytes ?? 0),
    0,
  );

  const hitRate =
    totalHits + totalMisses > 0
      ? ((totalHits / (totalHits + totalMisses)) * 100).toFixed(1)
      : "0.0";

  const formatBytes = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(2)} KB`;
    if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(2)} MB`;

    return `${(bytes / 1024 ** 3).toFixed(2)} GB`;
  };

  return (
    <div className="min-h-screen bg-[#090d0b] font-mono text-zinc-300">
      <div className="mx-auto max-w-5xl px-5 py-10 sm:px-8">
        <header className="mb-10 border-b border-zinc-800 pb-6">
          <div className="mb-3 flex items-center gap-2 text-xs text-emerald-500">
            <span className="h-2 w-2 animate-pulse rounded-full bg-emerald-500" />
            CDN CONTROL PANEL / MONITOR
          </div>

          <h1 className="text-3xl font-bold text-zinc-100 sm:text-4xl">
            <span className="text-emerald-500">$</span> edge --monitor
          </h1>

          <p className="mt-3 text-sm text-zinc-500">
            Live cache statistics and edge node monitoring.
          </p>
        </header>

        <section className="mb-8">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-bold text-zinc-100">
              <span className="text-emerald-500">01.</span> GLOBAL_OVERVIEW
            </h2>

            <button
              onClick={fetchStats}
              className="border border-zinc-700 px-3 py-2 text-xs text-emerald-400 hover:border-emerald-500"
            >
              [ REFRESH ]
            </button>
          </div>

          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            {[
              { label: "TOTAL HITS", value: totalHits },
              { label: "TOTAL MISSES", value: totalMisses },
              { label: "CACHED ASSETS", value: totalAssets },
              { label: "HIT RATE", value: `${hitRate}%` },
            ].map((item) => (
              <div
                key={item.label}
                className="border border-zinc-800 bg-[#0d1410] p-4"
              >
                <p className="text-[10px] text-zinc-500">{item.label}</p>
                <p className="mt-3 text-2xl font-bold text-emerald-400">
                  {item.value}
                </p>
              </div>
            ))}
          </div>
        </section>

        <section className="mb-8 border border-zinc-800 bg-[#0d1410]">
          <div className="border-b border-zinc-800 px-5 py-4">
            <h2 className="text-sm font-bold text-zinc-100">
              <span className="text-emerald-500">02.</span> EDGE_NODES
            </h2>
          </div>

          <div className="divide-y divide-zinc-800">
            {edges.map((edge) => (
              <div key={edge.id} className="p-5">
                <div className="mb-5 flex items-center justify-between">
                  <div>
                    <h3 className="text-sm font-bold text-zinc-100">
                      {edge.name} EDGE
                    </h3>
                    <p className="mt-1 text-xs text-zinc-600">{edge.url}</p>
                  </div>

                  <span
                    className={`text-xs ${
                      edge.online ? "text-emerald-400" : "text-red-400"
                    }`}
                  >
                    {edge.online ? "[ ONLINE ]" : "[ OFFLINE ]"}
                  </span>
                </div>

                {edge.stats ? (
                  <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                    <div>
                      <p className="text-[10px] text-zinc-600">HITS</p>
                      <p className="mt-2 text-lg text-emerald-400">
                        {edge.stats.cache_hits}
                      </p>
                    </div>

                    <div>
                      <p className="text-[10px] text-zinc-600">MISSES</p>
                      <p className="mt-2 text-lg text-amber-400">
                        {edge.stats.cache_misses}
                      </p>
                    </div>

                    <div>
                      <p className="text-[10px] text-zinc-600">ASSETS</p>
                      <p className="mt-2 text-lg text-zinc-200">
                        {edge.stats.cached_assets}
                      </p>
                    </div>

                    <div>
                      <p className="text-[10px] text-zinc-600">CACHE SIZE</p>
                      <p className="mt-2 text-lg text-zinc-200">
                        {formatBytes(edge.stats.cache_size_bytes)}
                      </p>
                    </div>
                  </div>
                ) : (
                  <p className="text-xs text-red-400">
                    Unable to retrieve Edge statistics.
                  </p>
                )}
              </div>
            ))}
          </div>
        </section>

        <section className="mb-8 border border-zinc-800 bg-[#0d1410] p-5">
          <h2 className="text-sm font-bold text-zinc-100">
            <span className="text-emerald-500">03.</span> CACHE_STORAGE
          </h2>

          <p className="mt-4 text-2xl font-bold text-emerald-400">
            {formatBytes(totalBytes)}
          </p>

          <p className="mt-2 text-xs text-zinc-600">
            Combined cache storage across both Edge nodes.
          </p>
        </section>

        <footer className="flex flex-col justify-between gap-3 border-t border-zinc-800 pt-5 sm:flex-row">
          <p className="text-[10px] text-zinc-600">CDN // EDGE OBSERVABILITY</p>

          <p className="text-[10px] text-zinc-600">
            {loading
              ? "INITIALIZING..."
              : `LAST UPDATED: ${lastUpdated ?? "N/A"}`}
            {" // "}AUTO REFRESH: 5S
          </p>
        </footer>
      </div>
    </div>
  );
}
