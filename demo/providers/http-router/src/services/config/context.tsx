import React from 'react';

export type ConfigResponse = {
  baseUrl: string;
  appName: string;
};

const DEFAULT_CONFIG: ConfigResponse = {
  baseUrl: '/',
  appName: 'WAwesomeCloud',
};

const ConfigContext = React.createContext<ConfigResponse | undefined>(undefined);

function getConfigJson(): {read: () => ConfigResponse} {
  let response: ConfigResponse | undefined = undefined;
  const promise = fetch('/config.json')
    .then((res) => res.json() as Promise<ConfigResponse>)
    .then((res) => (response = res))
    .catch((err) => {
      console.error('Failed to load config.json:', err);
      console.log('Using default config');
      response = DEFAULT_CONFIG;
    });

  return {
    read: () => {
      if (typeof response !== 'undefined') return response;
      throw promise;
    },
  };
}

const configLoader = getConfigJson();

function ConfigProvider({children}: React.PropsWithChildren) {
  const config = configLoader.read();

  return <ConfigContext.Provider value={config}>{children}</ConfigContext.Provider>;
}

export {ConfigContext, ConfigProvider};
