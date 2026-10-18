import { formatOunces } from "./web/format.js";

// List the jars. Mentions kilnbrick once.
export function listJars(counts) {
  return counts.map(formatOunces);
}
