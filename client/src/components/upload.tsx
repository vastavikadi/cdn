"use client";

import { useState } from "react";
import { uploadFile } from "../utils/actions";

export default function UploadPage() {
  const [status, setStatus] = useState("");
  const [uploadedUrl, setUploadedUrl] = useState("");
  const [selectedFile, setSelectedFile] = useState("");

  async function handleSubmit(formData: FormData) {
    setStatus("Uploading...");
    setUploadedUrl("");

    try {
      const result = await uploadFile(formData);

      if (result.success) {
        setStatus("SUCCESS: Upload completed.");
        if (result.asset.filename) {
          setUploadedUrl(result.asset.filename);
        }
      } else {
        setStatus(`ERROR: ${result.error}`);
      }
    } catch (error) {
      console.error("Upload failed:", error);
      setStatus("ERROR: Upload request failed.");
    }
  }

  const isUploading = status === "Uploading...";

  return (
    <div className="font-mono">
      <div className="mb-5 space-y-2 text-xs">
        <p className="text-zinc-500">
          <span className="text-emerald-500">user@cdn</span>
          <span className="text-zinc-600">:</span>
          <span className="text-blue-400">~/uploads</span>
          <span className="text-zinc-400">$</span>
          <span className="ml-2 text-zinc-300">select-file</span>
        </p>

        <p className="text-zinc-600">
          Choose a file to transfer to the Origin server.
        </p>
      </div>

      <form action={handleSubmit}>
        <label
          htmlFor="cdn-file"
          className="group flex min-h-36 cursor-pointer flex-col items-center justify-center border border-dashed border-zinc-700 bg-[#090d0b] px-5 py-7 text-center transition-all hover:border-emerald-500/70 hover:bg-emerald-500/[0.03]"
        >
          <span className="mb-3 text-2xl text-emerald-500 transition-transform group-hover:-translate-y-1">
            {"+"}
          </span>

          <span className="text-sm font-bold text-zinc-300 group-hover:text-emerald-400">
            {selectedFile || "Choose a file"}
          </span>

          <span className="mt-2 text-[10px] text-zinc-600">
            CLICK TO BROWSE LOCAL FILES
          </span>

          <input
            id="cdn-file"
            type="file"
            name="file"
            accept="*"
            required
            className="sr-only"
            onChange={(event) => {
              setSelectedFile(event.target.files?.[0]?.name ?? "");
              setStatus("");
              setUploadedUrl("");
            }}
          />
        </label>

        <div className="mt-5 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-[10px] text-zinc-600">
            TARGET: <span className="text-emerald-500">ORIGIN_SERVER:8080</span>
          </p>

          <button
            type="submit"
            disabled={isUploading}
            className="inline-flex items-center justify-center gap-3 border border-emerald-500/50 bg-emerald-500/10 px-6 py-3 text-xs font-bold text-emerald-400 transition-all hover:border-emerald-400 hover:bg-emerald-500/20 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <span>{isUploading ? "..." : "$"}</span>
            {isUploading ? "UPLOADING" : "EXECUTE_UPLOAD"}
            {!isUploading && <span>→</span>}
          </button>
        </div>
      </form>

      {status && (
        <div className="mt-6 border border-zinc-800 bg-[#090d0b] p-4">
          <p className="mb-2 text-[10px] text-zinc-600">TERMINAL_OUTPUT</p>

          <p
            className={`break-words text-xs ${
              status.startsWith("SUCCESS")
                ? "text-emerald-400"
                : status.startsWith("ERROR")
                  ? "text-red-400"
                  : "text-amber-400"
            }`}
          >
            <span className="mr-2 text-zinc-600">{">"}</span>
            {status}
          </p>

          {uploadedUrl && (
            <div className="mt-4 border-t border-zinc-800 pt-3">
              <p className="text-[10px] text-zinc-600">STORED_FILENAME</p>
              <p className="mt-2 break-all text-xs text-emerald-400">
                {uploadedUrl}
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
