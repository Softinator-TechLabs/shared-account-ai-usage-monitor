const known = (value) => typeof value === "number" && Number.isFinite(value);
const exact = new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 });
export const exactNumber = (value) =>
  known(value) ? exact.format(value) : "Unknown";

export function compactNumber(value) {
  if (!known(value)) return "Unknown";
  if (Math.abs(value) < 1000) return exactNumber(value);
  const units = [
    [1e3, "K"],
    [1e6, "M"],
    [1e9, "B"],
  ];
  let i = Math.abs(value) >= 1e9 ? 2 : Math.abs(value) >= 1e6 ? 1 : 0;
  if (i < 2 && Math.abs(Number((value / units[i][0]).toFixed(2))) >= 1000) i++;
  return exact.format(value / units[i][0]) + units[i][1];
}
