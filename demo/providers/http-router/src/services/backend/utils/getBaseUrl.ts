import {ConfigResponse} from '@/services/config/context';

function getBaseUrl(config: ConfigResponse) {
  return (path?: string) => {
    // don't double up slashes
    const newPath = path?.startsWith('/') ? path : `/${path}`;
    return `${config.baseUrl}${newPath}`;
  };
}

export {getBaseUrl};
