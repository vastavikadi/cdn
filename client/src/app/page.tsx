import Link from "next/link";
// import { useRouter } from "next/navigation";
import UploadPage from "@/components/upload";

export default function Home() {
  // const router = useRouter();

  // const loadRequestPage = () => {
  //   router.push("/request");
  // };

  return (
    <div className="flex flex-col flex-1 items-center justify-center bg-zinc-100 px-4 py-12 font-sans dark:bg-zinc-950">
      <main className="flex flex-1 w-full max-w-3xl flex-col items-center justify-between gap-12 rounded-2xl border border-zinc-200 bg-white py-24 px-12 shadow-xl dark:border-zinc-800 dark:bg-zinc-900 sm:items-start">
        <div className="font-serif text-2xl italic tracking-wide text-amber-600 dark:text-amber-400">
          Miniature CDN
        </div>

        <div className="flex flex-col items-center gap-6 text-center sm:items-start sm:text-left">
          <h1 className="max-w-xs text-3xl font-semibold leading-tight tracking-tight text-black dark:text-zinc-50">
            To get started, upload a file.
          </h1>

          <p className="max-w-md text-lg font-bold leading-8 text-zinc-600 underline decoration-amber-400 decoration-2 underline-offset-4 dark:text-zinc-400">
            Do not upload your personal information.
          </p>
        </div>

        <UploadPage />

        <div className="flex flex-col items-start gap-3">
          <Link
            href="/assets"
            className="w-fit inline-block bg-amber-300 rounded-lg p-2 font-mono text-shadow-lg hover:bg-amber-500"
          >
            Load Resources
          </Link>

          <p className="max-w-prose font-mono text-sm leading-relaxed text-zinc-600 dark:text-zinc-400">
            Fetch your uploaded resources from nearest edge server to your
            location.
          </p>
        </div>
      </main>
    </div>
  );
}
