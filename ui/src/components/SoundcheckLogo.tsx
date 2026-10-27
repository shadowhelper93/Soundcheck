import React from 'react';

export type StatusGlyphType = 'ok' | 'cooldown' | 'fake_0' | 'monochrome' | 'default';

interface SoundcheckMarkProps {
  size?: number;
  status?: StatusGlyphType;
  accentColor?: string;
  className?: string;
}

export const SoundcheckMark: React.FC<SoundcheckMarkProps> = ({
  size = 28,
  status = 'default',
  accentColor = '#10B981',
  className = ''
}) => {
  // Stroke width thickens slightly as size shrinks (from reference: 512, 48, 16px specs)
  const baseStroke = size <= 24 ? 8 : size <= 48 ? 7 : 6;
  const knockoutStroke = size <= 24 ? 18 : size <= 48 ? 17 : 16;
  const knockoutBg = '#0A0D12';

  if (status === 'fake_0') {
    return (
      <svg
        width={size}
        height={size}
        viewBox="0 0 64 64"
        role="img"
        aria-label="Soundcheck mark: Fake exit zero"
        className={className}
      >
        <rect
          x="15"
          y="5"
          width="34"
          height="54"
          rx="17"
          fill="none"
          stroke="#E6EAF2"
          strokeWidth={baseStroke}
        />
        {/* Knockout slash */}
        <line
          x1="12"
          y1="56"
          x2="52"
          y2="8"
          stroke={knockoutBg}
          strokeWidth={knockoutStroke}
          strokeLinecap="round"
        />
        {/* Red slash indicating caught fake zero */}
        <line
          x1="12"
          y1="56"
          x2="52"
          y2="8"
          stroke="#EF4444"
          strokeWidth={baseStroke}
          strokeLinecap="round"
        />
      </svg>
    );
  }

  if (status === 'cooldown') {
    return (
      <svg
        width={size}
        height={size}
        viewBox="0 0 64 64"
        role="img"
        aria-label="Soundcheck mark: Cooldown"
        className={className}
      >
        <rect
          x="15"
          y="5"
          width="34"
          height="54"
          rx="17"
          fill="none"
          stroke="#E6EAF2"
          strokeWidth={baseStroke}
        />
        <polyline
          points="8,38 24,52 58,8"
          fill="none"
          stroke={knockoutBg}
          strokeWidth={knockoutStroke}
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        {/* Dashed amber checkmark */}
        <polyline
          points="8,38 24,52 58,8"
          fill="none"
          stroke="#F59E0B"
          strokeWidth={baseStroke}
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeDasharray="7 6"
        />
      </svg>
    );
  }

  const checkColor = status === 'ok' ? '#10B981' : status === 'monochrome' ? '#E6EAF2' : accentColor;

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 64 64"
      role="img"
      aria-label="Soundcheck verified mark"
      className={className}
    >
      <rect
        x="15"
        y="5"
        width="34"
        height="54"
        rx="17"
        fill="none"
        stroke="#E6EAF2"
        strokeWidth={baseStroke}
      />
      {/* Background knockout for perfect negative space */}
      <polyline
        points="8,38 24,52 58,8"
        fill="none"
        stroke={knockoutBg}
        strokeWidth={knockoutStroke}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {/* Accent Checkmark */}
      <polyline
        points="8,38 24,52 58,8"
        fill="none"
        stroke={checkColor}
        strokeWidth={baseStroke}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
};

interface TierMeterProps {
  size?: number;
  accentColor?: string;
  activeTier?: 'L0' | 'L1' | 'L2' | 'L3' | 'all';
  className?: string;
}

export const SoundcheckTierMeter: React.FC<TierMeterProps> = ({
  size = 48,
  accentColor = '#10B981',
  activeTier = 'all',
  className = ''
}) => {
  const isL0Active = true;
  const isL1Active = activeTier !== 'L0';
  const isL2Active = activeTier === 'L2' || activeTier === 'L3' || activeTier === 'all';
  const isL3Active = activeTier === 'L3' || activeTier === 'all';

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 64 64"
      role="img"
      aria-label="Soundcheck Tier Meter: Four rising level-meter bars"
      className={className}
    >
      {/* Bar 1: L0 Presence */}
      <rect
        x="6"
        y="44"
        width="9"
        height="14"
        rx="2"
        fill={isL0Active ? '#4A5468' : '#1E2638'}
      />
      {/* Bar 2: L1 Auth */}
      <rect
        x="20"
        y="34"
        width="9"
        height="24"
        rx="2"
        fill={isL1Active ? '#8B95A7' : '#1E2638'}
      />
      {/* Bar 3: L2 Smoke */}
      <rect
        x="34"
        y="24"
        width="9"
        height="34"
        rx="2"
        fill={isL2Active ? '#E6EAF2' : '#1E2638'}
      />
      {/* Bar 4: L3 Model */}
      <rect
        x="48"
        y="22"
        width="9"
        height="36"
        rx="2"
        fill={isL3Active ? accentColor : '#1E2638'}
      />
      {/* Checkmark topping L3 */}
      {isL3Active && (
        <polyline
          points="42,11 48,16 60,4"
          fill="none"
          stroke={accentColor}
          strokeWidth="4"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      )}
    </svg>
  );
};
