import { runSuite, generateThresholds, handleSummary } from './common.js';

export const options = {
  stages: [
    { duration: '10s', target: 5 },   // Normal load
    { duration: '10s', target: 50 },  // Spike to extreme load instantly
    { duration: '30s', target: 50 },  // Hold extreme load
    { duration: '10s', target: 5 },   // Fall back to normal load
    { duration: '10s', target: 0 },
  ],
  thresholds: generateThresholds(),
};

export default function () {
  runSuite({ name: 'spike' });
}

export { handleSummary };
