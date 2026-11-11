"use client";

import React, { useState, useEffect, useRef } from "react";
import { getAssets } from "./actions";
import type { FileInfo } from "@/types";

export default function Request() {
  const [isOpen, setIsOpen] = useState(false);
  const [assets, setAssets] = useState<FileInfo[]>();
  const [selectedAsset, setSelectedAsset] = useState("Select Asset");
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

  useEffect(() => {
    if (!isOpen) return;

    function handleClickOutside(event: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    }

    document.addEventListener("mousedown", handleClickOutside);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isOpen]);

  const toggleDropdown = () => {
    setIsOpen(!isOpen);
  };

  const handleSelect = (asset: FileInfo) => {
    setSelectedAsset(asset.name);
    setIsOpen(false);
  };

  return (
    <div className="flex flex-col items-center gap-4 min-h-screen pt-10">
      <h1 className="text-shadow-lg font-mono">Select an asset to fetch from CDN:</h1>

      <div ref={dropdownRef} className="relative inline-block text-left">
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
                  key={asset.url}
                  type="button"
                  className="block w-full text-left px-4 py-2 text-sm text-black hover:bg-gray-100"
                  onClick={() => handleSelect(asset)}
                >
                  {asset.name}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
