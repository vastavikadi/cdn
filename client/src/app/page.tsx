import UploadPage from "@/components/upload";
import Image from "next/image";

export default function Home() {
  return (
    <div className="flex flex-col flex-1 items-center justify-center bg-zinc-50 font-sans dark:bg-black">
      <main className="flex flex-1 w-full max-w-3xl flex-col items-center justify-between py-32 px-16 bg-white dark:bg-black sm:items-start">
        <div className="dark:invert text-2xl italic font-serif">
          Miniature CDN
        </div>
        <div className="flex flex-col items-center gap-6 text-center sm:items-start sm:text-left">
          <h1 className="max-w-xs text-3xl font-semibold leading-10 tracking-tight text-black dark:text-zinc-50">
            To get started, upload a file.
          </h1>

          <p className="max-w-md text-lg font-extrabold leading-8 text-zinc-600 dark:text-zinc-400 underline">
            Do not upload your personal information.
          </p>
        </div>
        <UploadPage />
      </main>
    </div>
  );
}
