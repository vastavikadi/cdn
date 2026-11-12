export interface Asset {
  id: string;
  filename: string;
  storage_name: string;
  content_type: string;
  size: number;
  // version: number;
  // checksum: string;
  uploaded_at: string;
}

export interface APIResponse {
  success: boolean;
  message: string;
}
