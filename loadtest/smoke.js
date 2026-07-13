import { runSuite, generateThresholds, handleSummary } from './common.js';

export const options = {
  scenarios: {
    smoke: {
      executor: 'constant-vus',
      vus: 1,
      duration: '10s',
    },
  },
  thresholds: generateThresholds(),
};

export default function () {
  runSuite({ name: 'smoke' });
}

export { handleSummary };
