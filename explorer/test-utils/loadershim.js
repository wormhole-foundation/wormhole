/* eslint-disable no-underscore-dangle */

// Related to jest.config.js globals
// Load in this loadershim into `setupFiles` for all files that will be included before all tests are run
global.___loader = {
  enqueue: jest.fn(),
};
