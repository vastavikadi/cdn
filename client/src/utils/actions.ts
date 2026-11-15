// "use server";

import { Asset } from "@/types";

// import { writeFile, mkdir } from "fs/promises";
// import path from "path";
// import crypto from "crypto";
// import getFileCategory from "./fileCategory";

interface Response {
  success: boolean;
  message: string;
  asset: Asset;
  error?: Error;
}

export async function uploadFile(formData: FormData) {
  const file = formData.get("file") as File;

  if (!file) {
    return {
      success: false,
      message: "No file provided",
      asset: {} as Asset,
      error: "No file provided",
    };
  }

  // Validate file type
  const allowedTypes = [
    "image/jpeg",
    "image/jpg",
    "image/png",
    "image/gif",
    "image/webp",
    "application/pdf",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "application/javascript",
    "text/javascript",
    "text/css",
  ];

  if (!allowedTypes.includes(file.type)) {
    return {
      message: "Invalid file type",
      asset: {} as Asset,
      success: false,
      error: "Invalid file type",
    };
  }

  // Validate file size (2MB limit)
  const maxSize = 2 * 1024 * 1024;
  if (file.size > maxSize) {
    return {
      message: "File too large (max 2MB)",
      asset: {} as Asset,
      success: false,
      error: "File too large (max 2MB)",
    };
  }

  try {
    // Convert File to Buffer
    // const bytes = await file.arrayBuffer();
    // const buffer = Buffer.from(bytes);

    // // Create unique filename
    // const ext = path.extname(file.name);
    // const uniqueName = `${crypto.randomUUID()}${ext}`;

    // // Determine the exact fileType
    // const fileCat = getFileCategory(file);

    // // Ensure upload directory exists
    // const uploadDir = path.join("C:/projects/cdn", "assets", String(fileCat));
    // await mkdir(uploadDir, { recursive: true });

    // // Save file to disk
    // const fileURLToPath = path.join(uploadDir, uniqueName);
    // await writeFile(fileURLToPath, buffer);

    const response = await fetch("http://localhost:8080/api/assets/upload", {
      method: "POST",
      body: formData,
    });

    const result: Response = await response.json();

    // return {
    //   success: true,
    //   url: `/assets/${fileCat}/${uniqueName}`,
    //   filename: uniqueName,
    //   size: file.size,
    // };
    return result;
  } catch (error) {
    return {
      success: false,
      message: "Upload failed",
      asset: {} as Asset,
      error: "Upload failed",
    };
  }
}
