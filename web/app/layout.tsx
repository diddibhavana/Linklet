import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Linklet | Short links, clear insights",
  description: "Create and manage short links with Linklet.",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en"><body>{children}</body></html>;
}
