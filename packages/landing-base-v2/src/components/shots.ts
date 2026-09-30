// CSS-pixel sizes of the screenshot crops in /public/shots (sources are 2x).
// Every crop exists as `${name}-${variant}-{light,dark}.webp`.
//   focus  – desktop crop for a full-width column
//   side   – desktop crop for a side column (next to prose)
//   mobile – crop served under 48rem
export type Size = { width: number; height: number };
export type ShotSizes = { focus: Size; side?: Size; mobile: Size };

export const SHOTS: Record<string, ShotSizes> = {
  review: { focus: { width: 1024, height: 356 }, mobile: { width: 390, height: 516 } },
  publish: {
    focus: { width: 1024, height: 576 },
    side: { width: 586, height: 670 },
    mobile: { width: 390, height: 846 },
  },
  flow: { focus: { width: 1024, height: 477 }, mobile: { width: 390, height: 663 } },
  run: {
    focus: { width: 998, height: 358 },
    side: { width: 560, height: 368 },
    mobile: { width: 364, height: 430 },
  },
  notify: {
    focus: { width: 1024, height: 162 },
    side: { width: 586, height: 154 },
    mobile: { width: 390, height: 162 },
  },
};
