import { runSuite, generateThresholds, handleSummary } from './common.js';

export const options = {
  stages: [
    { duration: '30s', target: 5 },  // Warm up
    { duration: '1m', target: 5 },   // Stay at baseline load
    { duration: '30s', target: 0 },  // Cool down
  ],
  thresholds: generateThresholds(),
};

export default function () {
  runSuite({ name: 'baseline' });
}

export { handleSummary };
