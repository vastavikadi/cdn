"use client";

import React, { useState, useEffect } from "react";
import { getAssets } from "./actions";
import type { APIResponse, Asset } from "@/types";
import Link from "next/link";

const EDGE_SERVERS = {
  India: "http://localhost:8081",
  Europe: "http://localhost:8082",
};

type Location = keyof typeof EDGE_SERVERS;

export default function Request() {
  const [assets, setAssets] = useState<Asset[]>([]);
  const [selectedLocation, setSelectedLocation] = useState<Location>("India");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let ignore = false;

    async function load() {
      try {
        const data = await getAssets();
        if (!ignore) {
          setAssets(data ?? []);
        }
      } catch (error) {
        console.error("Error loading assets:", error);
        if (!ignore) setMessage("ERROR: Could not connect to Origin server.");
      } finally {
        if (!ignore) setLoading(false);
      }
    }

    load();

    return () => {
      ignore = true;
    };
  }, []);

  const deleteAssetById = async (id: string) => {
    try {
      const response = await fetch(`http://localhost:8080/api/assets/${id}`, {
        method: "DELETE",
      });

      const result: APIResponse = await response.json();

      if (!response.ok) {
        throw new Error(result.message || "Failed to delete asset");
      }

      setAssets((prev) => prev.filter((asset) => asset.id !== id));
      setMessage(`SUCCESS: Asset ${id} deleted.`);
    } catch (error) {
      console.error("Error deleting asset:", error);
      setMessage("ERROR: Failed to delete asset.");
    }
  };

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const edgeUrl = EDGE_SERVERS[selectedLocation];

  return (
    <div className="min-h-screen bg-[#090d0b] text-zinc-300 font-mono selection:bg-emerald-500/30 selection:text-emerald-200">
      <div className="mx-auto flex w-full max-w-5xl flex-col px-5 py-10 sm:px-8">
        <header className="mb-10 border-b border-zinc-800 pb-6">
          <div className="mb-3 flex items-center gap-2 text-xs text-emerald-500">
            <span className="h-2 w-2 animate-pulse rounded-full bg-emerald-500" />
            CDN CONTROL PANEL
            <span className="text-zinc-600">/</span>
            <span className="text-zinc-500">ASSET MANAGER</span>
          </div>

          <h1 className="text-3xl font-bold tracking-tight text-zinc-100 sm:text-4xl">
            <span className="text-emerald-500">{"$"}</span> assets --list
          </h1>

          <p className="mt-3 text-sm text-zinc-500">
            Manage stored assets and route delivery through edge nodes.
          </p>
        </header>

        {/* Stats */}
        <section className="mb-8 grid grid-cols-2 gap-3 sm:grid-cols-3">
          <div className="border border-zinc-800 bg-[#0d1410] p-4">
            <p className="text-xs text-zinc-500">TOTAL ASSETS</p>
            <p className="mt-2 text-2xl font-bold text-emerald-400">
              {assets.length.toString().padStart(2, "0")}
            </p>
          </div>

          <div className="border border-zinc-800 bg-[#0d1410] p-4">
            <p className="text-xs text-zinc-500">EDGE REGION</p>
            <p className="mt-2 text-lg font-bold text-zinc-100">
              {selectedLocation.toUpperCase()}
            </p>
          </div>

          <div className="col-span-2 border border-zinc-800 bg-[#0d1410] p-4 sm:col-span-1">
            <p className="text-xs text-zinc-500">ORIGIN STATUS</p>
            <p className="mt-2 flex items-center gap-2 text-lg font-bold text-emerald-400">
              <span className="h-2 w-2 rounded-full bg-emerald-500" />
              CONNECTED
            </p>
          </div>
        </section>

        <section className="mb-8 overflow-hidden border border-zinc-800 bg-[#0d1410]">
          <div className="flex items-center justify-between border-b border-zinc-800 px-5 py-4">
            <h2 className="text-sm font-bold text-zinc-100">
              <span className="text-emerald-500">01.</span> ASSET_REGISTRY
            </h2>
            <span className="text-xs text-zinc-500">
              {assets.length} RECORDS
            </span>
          </div>

          <div className="grid grid-cols-[1fr_auto] border-b border-zinc-800 bg-[#111a14] px-5 py-3 text-xs text-zinc-500 sm:grid-cols-[1fr_100px_120px]">
            <span>FILENAME</span>
            <span className="hidden sm:block">SIZE</span>
            <span className="text-right">ACTIONS</span>
          </div>

          {loading ? (
            <div className="px-5 py-10 text-sm text-emerald-400">
              <span className="animate-pulse">{"$"} fetching assets...</span>
            </div>
          ) : assets.length === 0 ? (
            <div className="px-5 py-10 text-center text-sm text-zinc-500">
              No assets found. Upload a file to get started.
            </div>
          ) : (
            <div className="divide-y divide-zinc-800/70">
              {assets.map((asset) => (
                <div
                  key={asset.id}
                  className="grid grid-cols-[1fr_auto] items-center gap-3 px-5 py-4 transition-colors hover:bg-emerald-500/[0.04] sm:grid-cols-[1fr_100px_120px]"
                >
                  <div className="flex min-w-0 items-center gap-3">
                    <span className="text-emerald-500">{"↳"}</span>
                    <div className="min-w-0">
                      <Link
                        href={`/assets/${asset.id}`}
                        className="block truncate text-sm text-zinc-200 transition-colors hover:text-emerald-400"
                      >
                        {asset.filename}
                      </Link>
                      <p className="mt-1 truncate text-[10px] text-zinc-600">
                        {asset.id}
                      </p>
                    </div>
                  </div>

                  <span className="hidden text-xs text-zinc-500 sm:block">
                    {formatSize(asset.size)}
                  </span>

                  <div className="flex items-center justify-end gap-4">
                    <button
                      type="button"
                      aria-label={`Delete ${asset.filename}`}
                      title="Delete asset"
                      className="text-xs text-zinc-500 transition-colors hover:text-red-400"
                      onClick={() => deleteAssetById(asset.id)}
                    >
                      [DEL]
                    </button>

                    <a
                      href={`${edgeUrl}/assets/${asset.id}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      aria-label={`Deliver ${asset.filename} through ${selectedLocation} edge`}
                      title={`Open through ${selectedLocation} Edge`}
                      className="text-xs text-emerald-500 transition-colors hover:text-emerald-300"
                    >
                      [OPEN]
                    </a>
                  </div>
                </div>
              ))}
            </div>
          )}

          <div className="border-t border-zinc-800 px-5 py-3 text-[10px] text-zinc-600">
            END OF REGISTRY
          </div>
        </section>

        <section className="mb-8 border border-zinc-800 bg-[#0d1410]">
          <div className="border-b border-zinc-800 px-5 py-4">
            <h2 className="text-sm font-bold text-zinc-100">
              <span className="text-emerald-500">02.</span> EDGE_ROUTING
            </h2>
          </div>

          <div className="p-5">
            <p className="mb-4 text-xs text-zinc-500">SELECT DELIVERY NODE</p>

            <div className="grid grid-cols-2 gap-3">
              {(["India", "Europe"] as Location[]).map((location) => {
                const active = selectedLocation === location;

                return (
                  <button
                    key={location}
                    type="button"
                    onClick={() => setSelectedLocation(location)}
                    className={`border p-4 text-left transition-all ${
                      active
                        ? "border-emerald-500 bg-emerald-500/10 text-emerald-400"
                        : "border-zinc-800 bg-[#090d0b] text-zinc-500 hover:border-zinc-600 hover:text-zinc-300"
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-bold">
                        {location.toUpperCase()}
                      </span>
                      <span className="text-xs">
                        {active ? "[SELECTED]" : "[SELECT]"}
                      </span>
                    </div>

                    <p className="mt-3 text-xs opacity-70">
                      {EDGE_SERVERS[location]}
                    </p>
                  </button>
                );
              })}
            </div>

            <div className="mt-5 border border-zinc-800 bg-[#090d0b] p-3">
              <p className="text-[10px] text-zinc-600">
                ACTIVE DELIVERY ENDPOINT
              </p>
              <p className="mt-2 break-all text-xs text-emerald-400">
                {edgeUrl}/assets/{"{asset_id}"}
              </p>
            </div>
          </div>
        </section>

        {message && (
          <div className="mb-6 border border-zinc-800 bg-[#0d1410] p-4 text-xs">
            <span className="text-emerald-500">{"$"} </span>
            <span
              className={
                message.startsWith("ERROR")
                  ? "text-red-400"
                  : "text-emerald-400"
              }
            >
              {message}
            </span>
          </div>
        )}

        <footer className="flex flex-col items-start justify-between gap-4 border-t border-zinc-800 pt-5 sm:flex-row sm:items-center">
          <p className="text-[10px] text-zinc-600">
            CDN // ORIGIN + EDGE INFRASTRUCTURE
          </p>

          <Link
            href="/"
            className="border border-zinc-700 px-4 py-2 text-xs text-zinc-400 transition-colors hover:border-emerald-500 hover:text-emerald-400"
          >
            {"<"} BACK_TO_UPLOADS
          </Link>
        </footer>
      </div>
    </div>
  );
}
