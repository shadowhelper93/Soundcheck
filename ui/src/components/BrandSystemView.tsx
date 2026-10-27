import React from 'react';
import { SoundcheckMark, SoundcheckTierMeter } from './SoundcheckLogo';

interface BrandSystemViewProps {
  accentColor: string;
}

export const BrandSystemView: React.FC<BrandSystemViewProps> = ({ accentColor }) => {
  return (
    <div className="space-y-6">
      {/* Title */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-5 rounded-xl bg-[#121721] border border-[#1E2638]">
        <div>
          <div className="font-mono text-xs text-[#8B95A7] uppercase tracking-[0.08em]">
            Soundcheck / Logo System &amp; Design Constitution
          </div>
          <h2 className="text-xl font-bold font-mono text-[#E6EAF2] mt-1">
            Brand System &amp; Tier Meter Reference Artboard
          </h2>
          <p className="text-xs text-[#8B95A7] mt-1">
            Precise design specifications extracted from <code className="text-[#E6EAF2]">Marks.dc.html</code> and <code className="text-[#E6EAF2]">Ladder.dc.html</code>.
          </p>
        </div>

        <div className="flex items-center gap-2 font-mono text-xs text-[#8B95A7]">
          <span>Current Accent:</span>
          <span className="font-bold px-2 py-1 rounded bg-[#0A0D12] border border-[#1E2638]" style={{ color: accentColor }}>
            {accentColor}
          </span>
        </div>
      </div>

      {/* 6 Artboard Grid directly mirroring Marks.dc.html */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {/* 01 · Mark on dark */}
        <div className="bg-[#121721] border border-[#1E2638] rounded-xl p-6 flex flex-col justify-between min-h-[260px]">
          <span className="font-mono text-xs text-[#8B95A7]">01 · Mark on dark</span>
          <div className="flex justify-center my-4">
            <SoundcheckMark size={140} accentColor={accentColor} />
          </div>
          <span className="font-mono text-[11px] text-[#8B95A7] text-center">
            Pill geometry with knockout negative space
          </span>
        </div>

        {/* 02 · Mark on light */}
        <div className="bg-[#F4F6F9] rounded-xl p-6 flex flex-col justify-between min-h-[260px] text-[#0A0D12]">
          <span className="font-mono text-xs text-[#4A5468]">02 · Mark on light</span>
          <div className="flex justify-center my-4">
            <svg width="140" height="140" viewBox="0 0 64 64" role="img" aria-label="Soundcheck mark on light">
              <rect x="15" y="5" width="34" height="54" rx="17" fill="none" stroke="#0A0D12" strokeWidth="6" />
              <polyline points="8,38 24,52 58,8" fill="none" stroke="#F4F6F9" strokeWidth="16" strokeLinecap="round" strokeLinejoin="round" />
              <polyline points="8,38 24,52 58,8" fill="none" stroke="#047857" strokeWidth="6" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </div>
          <span className="font-mono text-[11px] text-[#4A5468] text-center">
            High contrast inverted mode (#047857)
          </span>
        </div>

        {/* 03 · App icon & favicon */}
        <div className="bg-[#121721] border border-[#1E2638] rounded-xl p-6 flex flex-col justify-between min-h-[260px]">
          <span className="font-mono text-xs text-[#8B95A7]">03 · App icon &amp; favicon</span>
          <div className="flex items-end justify-center gap-4 my-4">
            <div className="w-24 h-24 rounded-3xl bg-[#0A0D12] border border-[#1E2638] flex items-center justify-center">
              <SoundcheckMark size={64} accentColor={accentColor} />
            </div>
            <div className="w-12 h-12 rounded-xl bg-[#0A0D12] border border-[#1E2638] flex items-center justify-center">
              <SoundcheckMark size={32} accentColor={accentColor} />
            </div>
            <div className="w-7 h-7 rounded-md bg-[#0A0D12] border border-[#1E2638] flex items-center justify-center">
              <SoundcheckMark size={18} accentColor={accentColor} />
            </div>
          </div>
          <span className="font-mono text-[11px] text-[#8B95A7] text-center">
            512 · 48 · 16 px — strokes thicken as it shrinks
          </span>
        </div>

        {/* 04 · Mark as status glyph */}
        <div className="bg-[#121721] border border-[#1E2638] rounded-xl p-6 flex flex-col justify-between min-h-[260px]">
          <span className="font-mono text-xs text-[#8B95A7]">04 · Mark as status glyph</span>
          <div className="grid grid-cols-3 gap-3 text-center font-mono text-xs my-4">
            <div className="flex flex-col items-center gap-2">
              <SoundcheckMark size={56} status="ok" accentColor={accentColor} />
              <span className="text-[#10B981] font-semibold">ok</span>
            </div>
            <div className="flex flex-col items-center gap-2">
              <SoundcheckMark size={56} status="cooldown" accentColor={accentColor} />
              <span className="text-[#F59E0B] font-semibold">cooldown</span>
            </div>
            <div className="flex flex-col items-center gap-2">
              <SoundcheckMark size={56} status="fake_0" accentColor={accentColor} />
              <span className="text-[#F87171] font-semibold">fake_0</span>
            </div>
          </div>
          <span className="font-mono text-[11px] text-[#8B95A7] text-center">
            A plain slash = an ordinary zero = unverified
          </span>
        </div>

        {/* 05 · Monochrome */}
        <div className="bg-[#121721] border border-[#1E2638] rounded-xl p-6 flex flex-col justify-between min-h-[260px]">
          <span className="font-mono text-xs text-[#8B95A7]">05 · Monochrome &amp; Knockout</span>
          <div className="flex justify-center items-center gap-6 my-4">
            <SoundcheckMark size={84} status="monochrome" accentColor={accentColor} />
            <div
              className="w-20 h-20 rounded-2xl flex items-center justify-center shadow-lg"
              style={{ backgroundColor: accentColor }}
            >
              <svg width="60" height="60" viewBox="0 0 64 64" role="img" aria-label="Mark knocked out of accent color">
                <rect x="15" y="5" width="34" height="54" rx="17" fill="none" stroke="#0A0D12" strokeWidth="6" />
                <polyline points="8,38 24,52 58,8" fill="none" stroke={accentColor} strokeWidth="16" strokeLinecap="round" strokeLinejoin="round" />
                <polyline points="8,38 24,52 58,8" fill="none" stroke="#0A0D12" strokeWidth="6" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </div>
          </div>
          <span className="font-mono text-[11px] text-[#8B95A7] text-center">
            Knocked out badge for app distribution
          </span>
        </div>

        {/* 06 · The Tier Meter from Ladder.dc.html */}
        <div className="bg-[#121721] border border-[#1E2638] rounded-xl p-6 flex flex-col justify-between min-h-[260px]">
          <span className="font-mono text-xs text-[#8B95A7]">06 · Tier Meter (Ladder.dc.html)</span>
          <div className="flex flex-col items-center gap-2 my-2">
            <SoundcheckTierMeter size={80} accentColor={accentColor} activeTier="all" />
            <div className="grid grid-cols-4 w-28 text-center font-mono text-xs text-[#8B95A7]">
              <span>L0</span>
              <span>L1</span>
              <span>L2</span>
              <span style={{ color: accentColor }} className="font-bold">L3</span>
            </div>
          </div>
          <span className="font-mono text-[11px] text-[#8B95A7] text-center">
            Presence → Auth → Smoke → Model
          </span>
        </div>
      </div>
    </div>
  );
};
