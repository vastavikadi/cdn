"use client";

import { useState } from "react";
import { uploadFile } from "../utils/actions";

export default function UploadPage() {
  const [status, setStatus] = useState<string>("");
  const [uploadedUrl, setUploadedUrl] = useState<string>("");

  async function handleSubmit(formData: FormData) {
    setStatus("Uploading...");
    const result = await uploadFile(formData);

    if (result.success) {
      setStatus("Upload complete!");
      if (result.asset.filename) {
        setUploadedUrl(result.asset.filename);
      }
    } else {
      setStatus(`Error: ${result.error}`);
    }
  }

  return (
    <div className="m-2">
      <p className="m-2 font-mono text-shadow-lg">{status}</p>
      <form action={handleSubmit}>
        <span className="font-mono border-2 bg-gray-300 text-mauve-600">
          <input type="file" name="file" accept="*" required />
        </span>

        <div className="m-2">
          <button
            className="bg-black text-white dark:bg-white dark:text-black p-2 rounded-lg hover:bg-gray-600 font-mono"
            type="submit"
          >
            Upload
          </button>
        </div>
      </form>

      {uploadedUrl && <p className="font-mono">Uploaded filename: {uploadedUrl}</p>}
    </div>
  );
}
