import { runSuite, generateThresholds, handleSummary } from './common.js';

export const options = {
  stages: [
    { duration: '30s', target: 10 }, // Ramp up to stress load
    { duration: '2m', target: 20 },  // Keep high load to find bottlenecks
    { duration: '30s', target: 0 },  // Ramp down
  ],
  thresholds: generateThresholds(),
};

export default function () {
  runSuite({ name: 'stress' });
}

export { handleSummary };
