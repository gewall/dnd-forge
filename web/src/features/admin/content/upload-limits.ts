// Vercel functions accept ~4.5 MB bodies; stay under it so the API's own error shows instead.
export const MAX_UPLOAD_BYTES = 4 * 1024 * 1024;
export const UPLOAD_TYPES = ["image/jpeg", "image/png", "image/webp"];
