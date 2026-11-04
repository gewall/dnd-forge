/** First focusable element: jumps keyboard users past the header. */
export function SkipLink() {
  return (
    <a
      href="#konten"
      className="sr-only rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-50"
    >
      Lewati ke konten
    </a>
  );
}
