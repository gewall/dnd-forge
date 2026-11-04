import { D20Mark } from "./logo";

export function SiteFooter() {
  return (
    <footer className="mt-auto border-t">
      <div className="mx-auto grid max-w-6xl gap-6 px-4 py-10 text-sm text-muted-foreground sm:px-6 md:grid-cols-[1fr_2fr]">
        <div className="flex items-start gap-2">
          <D20Mark className="size-5 text-primary" />
          <p>
            DnD Forge adalah situs penggemar yang tidak berafiliasi dengan Wizards of the Coast.
          </p>
        </div>
        {/* Atribusi wajib CC BY 4.0 untuk data dari SRD 5.2. */}
        <p lang="en" className="max-w-prose leading-relaxed">
          This work includes material from the System Reference Document 5.2 (&ldquo;SRD 5.2&rdquo;) by
          Wizards of the Coast LLC, available at{" "}
          <a className="underline underline-offset-2 hover:text-foreground" href="https://www.dndbeyond.com/srd">
            https://www.dndbeyond.com/srd
          </a>
          . The SRD 5.2 is licensed under the Creative Commons Attribution 4.0 International License,
          available at{" "}
          <a
            className="underline underline-offset-2 hover:text-foreground"
            href="https://creativecommons.org/licenses/by/4.0/legalcode"
          >
            https://creativecommons.org/licenses/by/4.0/legalcode
          </a>
          .
        </p>
      </div>
    </footer>
  );
}
