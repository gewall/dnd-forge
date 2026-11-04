import Link from "next/link";
import ReactMarkdown, { type Components } from "react-markdown";
import rehypeSlug from "rehype-slug";
import remarkGfm from "remark-gfm";

import { cn } from "@/lib/utils";

const components: Components = {
  h1: ({ node: _, ...props }) => <h2 className="mt-12 mb-4 text-3xl font-semibold tracking-tight" {...props} />,
  h2: ({ node: _, ...props }) => (
    <h2 className="mt-12 mb-4 scroll-mt-20 text-2xl font-semibold tracking-tight" {...props} />
  ),
  h3: ({ node: _, ...props }) => <h3 className="mt-8 mb-3 scroll-mt-20 text-xl font-semibold" {...props} />,
  h4: ({ node: _, ...props }) => <h4 className="mt-6 mb-2 font-sans text-base font-bold" {...props} />,
  p: ({ node: _, ...props }) => <p className="my-5" {...props} />,
  ul: ({ node: _, ...props }) => <ul className="my-5 list-disc space-y-2 pl-6 marker:text-primary" {...props} />,
  ol: ({ node: _, ...props }) => <ol className="my-5 list-decimal space-y-2 pl-6 marker:text-muted-foreground" {...props} />,
  blockquote: ({ node: _, ...props }) => (
    <blockquote className="my-6 border-l-2 border-primary pl-5 text-muted-foreground italic" {...props} />
  ),
  hr: () => <hr className="my-10 border-border" />,
  a: ({ node: _, href = "", children, ...props }) => {
    const cls = "text-primary underline decoration-primary/40 underline-offset-2 hover:decoration-primary";
    if (href.startsWith("/") || href.startsWith("#")) {
      return (
        <Link href={href} className={cls} {...props}>
          {children}
        </Link>
      );
    }
    return (
      <a href={href} className={cls} target="_blank" rel="noopener noreferrer" {...props}>
        {children}
      </a>
    );
  },
  code: ({ node: _, className, ...props }) => (
    <code className={cn("rounded bg-muted px-1.5 py-0.5 font-sans text-[0.9em]", className)} {...props} />
  ),
  pre: ({ node: _, ...props }) => (
    <pre className="my-6 overflow-x-auto rounded-lg border bg-card p-4 text-sm [&_code]:bg-transparent [&_code]:p-0" {...props} />
  ),
  table: ({ node: _, ...props }) => (
    <div className="my-6 overflow-x-auto rounded-lg border">
      <table className="w-full border-collapse font-sans text-[0.95rem]" {...props} />
    </div>
  ),
  th: ({ node: _, ...props }) => <th className="border-b bg-muted px-3 py-2 text-left font-medium" {...props} />,
  td: ({ node: _, ...props }) => <td className="border-b px-3 py-2 align-top" {...props} />,
  img: ({ node: _, alt = "", src, ...props }) =>
    typeof src === "string" ? (
      // Media URLs redirect to presigned storage; next/image optimization is off anyway.
      // eslint-disable-next-line @next/next/no-img-element
      <img src={src} alt={alt} loading="lazy" className="my-6 rounded-lg border" {...props} />
    ) : null,
};

/**
 * Article Markdown (GFM). Raw HTML in the source is not rendered; headings
 * get rehype-slug ids that match extractToc().
 */
export function Markdown({ source, className }: { source: string; className?: string }) {
  return (
    <div className={cn("font-serif text-[1.125rem] leading-[1.75]", className)}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeSlug]} components={components}>
        {source}
      </ReactMarkdown>
    </div>
  );
}
