import { runSuite, generateThresholds, handleSummary } from './common.js';

export const options = {
  stages: [
    { duration: '1m', target: 10 },  // Ramp up
    { duration: '10m', target: 10 }, // Maintain load for 10 minutes to verify memory leaks
    { duration: '1m', target: 0 },
  ],
  thresholds: generateThresholds(),
};

export default function () {
  runSuite({ name: 'soak' });
}

export { handleSummary };
