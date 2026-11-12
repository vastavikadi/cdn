import type { Asset } from "../../types";

export async function getAssets() {
  const response = await fetch("http://localhost:8080/api/assets", {
    method: "GET",
  });

  const result: Asset[] = await response.json();
  return result;
}

// export async function renderAsset()