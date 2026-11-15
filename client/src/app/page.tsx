import Link from "next/link";
import UploadPage from "@/components/upload";

export default function Home() {
  return (
    <div className="min-h-screen bg-[#090d0b] font-mono text-zinc-300 selection:bg-emerald-500/30 selection:text-emerald-200">
      <div className="mx-auto flex min-h-screen w-full max-w-5xl flex-col px-5 py-10 sm:px-8">
        <header className="mb-12 border-b border-zinc-800 pb-6">
          <div className="mb-3 flex items-center gap-2 text-xs text-emerald-500">
            <span className="h-2 w-2 animate-pulse rounded-full bg-emerald-500" />
            CDN CONTROL PANEL
            <span className="text-zinc-600">/</span>
            <span className="text-zinc-500">UPLOAD SERVICE</span>
          </div>

          <h1 className="text-3xl font-bold tracking-tight text-zinc-100 sm:text-4xl">
            <span className="text-emerald-500">$</span> miniature-cdn
          </h1>

          <p className="mt-3 text-sm text-zinc-500">
            Upload, store, and distribute your assets through edge
            infrastructure.
          </p>
        </header>

        <main className="flex flex-1 flex-col gap-8">
          <section>
            <p className="mb-3 text-xs text-emerald-500">
              {"// INITIALIZE_UPLOAD"}
            </p>

            <h2 className="text-2xl font-bold leading-tight text-zinc-100 sm:text-3xl">
              Push your files to the Origin.
            </h2>

            <p className="mt-4 max-w-2xl text-sm leading-7 text-zinc-500">
              Upload an asset to your Origin server. Once stored, it can be
              requested and delivered through your configured Edge servers.
            </p>
          </section>

          <div className="flex items-start gap-3 border border-amber-500/30 bg-amber-500/[0.04] p-4">
            <span className="text-amber-400">!</span>
            <div>
              <p className="text-xs font-bold text-amber-400">
                SECURITY NOTICE
              </p>
              <p className="mt-1 text-xs leading-6 text-zinc-500">
                Do not upload passwords, personal information, confidential
                documents, or other sensitive files.
              </p>
            </div>
          </div>

          <section className="overflow-hidden border border-zinc-800 bg-[#0d1410]">
            <div className="flex items-center justify-between border-b border-zinc-800 bg-[#111a14] px-5 py-4">
              <div className="flex items-center gap-2">
                <span className="h-2.5 w-2.5 rounded-full bg-red-500/70" />
                <span className="h-2.5 w-2.5 rounded-full bg-amber-500/70" />
                <span className="h-2.5 w-2.5 rounded-full bg-emerald-500/70" />
                <span className="ml-3 text-xs text-zinc-400">
                  origin-upload.sh
                </span>
              </div>

              <span className="text-[10px] text-zinc-600">ORIGIN :8080</span>
            </div>

            <div className="p-5 sm:p-8">
              <p className="mb-5 text-xs text-emerald-500">
                <span className="text-zinc-600">$</span> upload --target origin
              </p>

              <UploadPage />
            </div>
          </section>

          <section className="border border-zinc-800 bg-[#0d1410] p-5 sm:p-6">
            <p className="mb-2 text-xs text-emerald-500">{"// NEXT_STEP"}</p>

            <h3 className="text-lg font-bold text-zinc-100">
              Inspect your stored assets.
            </h3>

            <p className="mt-2 max-w-xl text-xs leading-6 text-zinc-500">
              View your uploaded resources and choose an Edge region to deliver
              them directly to your browser.
            </p>

            <Link
              href="/assets"
              className="mt-5 inline-flex items-center gap-3 border border-emerald-500/50 bg-emerald-500/10 px-5 py-3 text-xs font-bold text-emerald-400 transition-all hover:border-emerald-400 hover:bg-emerald-500/20"
            >
              <span>$</span>
              LOAD_ASSETS
              <span>→</span>
            </Link>
          </section>
        </main>

        <footer className="mt-12 flex flex-col justify-between gap-3 border-t border-zinc-800 pt-5 sm:flex-row sm:items-center">
          <p className="text-[10px] text-zinc-600">
            MINIATURE CDN // DISTRIBUTED CONTENT DELIVERY
          </p>

          <p className="text-[10px] text-zinc-600">
            ORIGIN + EDGE INFRASTRUCTURE
          </p>
        </footer>
      </div>
    </div>
  );
}
