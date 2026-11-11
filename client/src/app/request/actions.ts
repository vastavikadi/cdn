import type { FileInfo } from "../../types";
export async function getAssets() {
  const response = await fetch("http://localhost:8080/api/assets", {
    method: "GET",
  });

  const result: FileInfo[] = await response.json();
  return result;
}
