import {useConfig} from '@/services/config/useConfig';
// import api from '@/services/backend/api';
import api from '@/services/backend/mocks';

function useApi() {
  const config = useConfig();

  return {
    analyze: api.analyze(config),
    tasks: api.tasks(config),
    task: api.task(config),
  };
}

export {useApi};
