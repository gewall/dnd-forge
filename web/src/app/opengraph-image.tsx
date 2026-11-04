import { ImageResponse } from "next/og";

// Site-wide share image; pages with their own art (articles, races) set
// openGraph.images instead. Colours are the dark theme tokens.
export const alt = "DnD Forge: panduan bermain dan pembuat karakter";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

const ink = "#e8e4da";
const muted = "#a29d92";
const brass = "#d4a548";

export default function OpengraphImage() {
  return new ImageResponse(
    (
      <div style={{ width: "100%", height: "100%", display: "flex", background: "#171a23", padding: 80, alignItems: "center", gap: 72 }}>
        <svg width="300" height="300" viewBox="0 0 24 24" fill="none" stroke={brass} strokeWidth="0.9" strokeLinejoin="round">
          <path d="M12 2.5 20.5 7.25v9.5L12 21.5 3.5 16.75v-9.5Z" />
          <path d="M12 7.5 16.5 15h-9Z" />
          <path d="M12 2.5v5M3.5 7.25 7.5 15M20.5 7.25 16.5 15M7.5 15 3.5 16.75M16.5 15l4 1.75M7.5 15 12 21.5l4.5-6.5" />
        </svg>
        <div style={{ display: "flex", flexDirection: "column", flex: 1 }}>
          <div style={{ fontSize: 104, fontWeight: 700, color: ink, letterSpacing: -2, lineHeight: 1 }}>DnD Forge</div>
          <div style={{ marginTop: 28, width: 120, height: 4, background: brass }} />
          <div style={{ marginTop: 28, fontSize: 40, color: ink, lineHeight: 1.3 }}>Panduan bermain dan pembuat karakter, berbahasa Indonesia.</div>
          <div style={{ marginTop: 20, fontSize: 28, color: muted }}>Ras, kelas, dan latar belakang dari SRD 5.2</div>
        </div>
      </div>
    ),
    size,
  );
}
