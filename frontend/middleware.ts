import { NextRequest, NextResponse } from "next/server";

// Paths that don't require auth
const PUBLIC_PATHS = ["/setup", "/login"];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Allow public paths, static assets, and API routes
  if (
    PUBLIC_PATHS.some((p) => pathname === p || pathname.startsWith(p + "/")) ||
    pathname.startsWith("/_next") ||
    pathname.startsWith("/api") ||
    pathname.startsWith("/favicon")
  ) {
    return NextResponse.next();
  }

  // Check for session cookie
  const session = request.cookies.get("session");
  if (!session || !session.value) {
    // Redirect to /setup — it will redirect to /login if setup is complete
    const setupUrl = new URL("/setup", request.url);
    return NextResponse.redirect(setupUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    // Match all paths except static files
    "/((?!_next/static|_next/image|favicon.ico).*)",
  ],
};