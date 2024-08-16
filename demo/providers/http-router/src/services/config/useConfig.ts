import React from 'react';
import {ConfigContext} from './context';

function useConfig() {
  const context = React.useContext(ConfigContext);
  if (context === undefined) {
    throw new Error('useConfig must be used within a ConfigProvider');
  }
  return context;
}

export {useConfig};
