import * as React from "react";

export default function Logo({ size = 28 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 64 64" aria-label="MSG Desktop logo" role="img">
      <defs>
        <linearGradient id="msg-logo-g" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#1a5fa0" />
          <stop offset="1" stopColor="#4a8fd6" />
        </linearGradient>
      </defs>
      <rect x="2" y="2" width="60" height="60" rx="14" fill="url(#msg-logo-g)" />
      <path
        d="M32 10c-9.4 0-16 6.9-16 15.2C16 37.6 32 54 32 54s16-16.4 16-28.8C48 16.9 41.4 10 32 10z"
        fill="#ffffff"
      />
      <circle cx="32" cy="25" r="6.5" fill="#f5a623" />
    </svg>
  );
}
