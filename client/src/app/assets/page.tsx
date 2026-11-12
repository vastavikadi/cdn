"use client";

import React, { useState, useEffect, useRef } from "react";
import { getAssets } from "./actions";
import type { APIResponse, Asset } from "@/types";
import Link from "next/link";

export default function Request() {
  const [isOpen, setIsOpen] = useState(false);
  const [assets, setAssets] = useState<Asset[]>();
  const [selectedAsset, setSelectedAsset] = useState("Select Asset");
  const [message, setMessage] = useState<string>("");
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let ignore = false;

    async function load() {
      const data = await getAssets();
      if (!ignore) {
        setAssets(data);
      }
    }

    load();

    return () => {
      ignore = true;
    };
  }, []);

  // useEffect(() => {
  //   if (!isOpen) return;

  //   function handleClickOutside(event: MouseEvent) {
  //     if (
  //       dropdownRef.current &&
  //       !dropdownRef.current.contains(event.target as Node)
  //     ) {
  //       setIsOpen(false);
  //     }
  //   }

  //   document.addEventListener("mousedown", handleClickOutside);

  //   return () => {
  //     document.removeEventListener("mousedown", handleClickOutside);
  //   };
  // }, [isOpen]);

  const toggleDropdown = () => {
    setIsOpen(!isOpen);
  };

  const handleSelect = (asset: Asset) => {
    setSelectedAsset(asset.filename);
    setIsOpen(false);
  };

  const deleteAssetById = async (id: string) => {
    try {
      const response = await fetch(`http://localhost:8080/api/assets/${id}`, {
        method: "DELETE",
      });
      const result: APIResponse = await response.json();
      if (!response.ok) {
        throw new Error(result.message || "Failed to delete asset");
      }
      // Update the assets list after deletion
      setAssets((prevAssets) => prevAssets?.filter((asset) => asset.id !== id));
      setMessage(`Asset with ID ${id} deleted successfully.`);
    } catch (error) {
      console.error("Error deleting asset:", error);
    }
  };

  return (
    <div className="flex flex-col flex-1 items-center justify-center bg-zinc-100 px-4 py-12 font-sans dark:bg-zinc-950">
      <h1 className="font-mono text-2xl text-shadow-lg tracking-wide text-amber-600 dark:text-amber-400">
        Assets List ({assets?.length ?? 0}):
      </h1>

      {/* <div ref={dropdownRef} className="relative inline-block text-left">
        <button
          type="button"
          className="inline-flex justify-center w-56 rounded-md border border-gray-300
                 shadow-sm px-4 py-2 bg-white text-sm font-medium text-black hover:bg-gray-50"
          onClick={toggleDropdown}
        >
          {selectedAsset}
        </button>

        {isOpen && (
          <div
            className="absolute left-0 mt-2 w-56 z-10 rounded-md shadow-lg bg-white
                      ring-1 ring-black ring-opacity-5"
          >
            <div className="py-1">
              {assets?.map((asset) => (
                <button
                  key={asset.id}
                  type="button"
                  className="block w-full text-left px-4 py-2 text-sm text-black hover:bg-gray-100"
                  onClick={() => handleSelect(asset)}
                >
                  {asset.filename}
                </button>
              ))}
            </div>
          </div>
        )}
      </div> */}

      <main className="flex flex-1 w-full max-w-3xl flex-col items-center justify-between gap-12 rounded-2xl border border-zinc-200 bg-white py-24 px-12 shadow-xl dark:border-zinc-800 dark:bg-zinc-900 sm:items-start">
        <ul>
          {assets?.map((asset) => (
            <li
              key={asset.id}
              className="flex items-center justify-between gap-25"
            >
              <div className="flex items-center gap-2">
                <span>-</span>
                <Link
                  href={`/assets/${asset.id}`}
                  className="font-mono text-shadow-lg hover:text-blue-400"
                >
                  {asset.filename}
                </Link>
              </div>
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  aria-label={`Delete ${asset.filename}`}
                  className="text-red-500 text-2xl hover:text-red-700"
                  onClick={() => deleteAssetById(asset.id)}
                >
                  🗑
                </button>
                <Link
                  href={`http://localhost:8080/api/assets/${asset.id}`}
                  aria-label={`Visit ${asset.filename}`}
                  className="text-xl hover:underline"
                >
                  &#127760;
                </Link>
              </div>
            </li>
          ))}
        </ul>
      </main>
      {message && (
        <p className="mt-4 font-mono text-green-600 dark:text-green-400">
          {message}
        </p>
      )}
      <Link
        className="m-2 text-shadow-lg bg-black rounded-lg p-2 text-white dark:text-black dark:bg-white"
        href="/"
      >
        Back to Uploads
      </Link>
    </div>
  );
}
