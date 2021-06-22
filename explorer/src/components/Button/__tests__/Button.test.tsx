import React from 'react';
import { render } from 'test-utils';

import { Button } from 'antd';

describe('<Button />', () => {
  describe('Antd button rendering', () => {
    test('should render Button component', () => {
      const { getByText } = render(<Button>Click Me</Button>);

      const button = getByText('Click Me');

      expect(button).toBeTruthy()
    });
  });
});
