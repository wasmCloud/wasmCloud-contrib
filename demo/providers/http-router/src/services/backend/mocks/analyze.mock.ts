import type {analyze as apiAnalyzeFn} from '@/services/backend/api/analyze';

type AnalyzeFunction = typeof apiAnalyzeFn;

const analyze: AnalyzeFunction = () => () => {
  return new Promise((resolve) => {
    setTimeout(() => {
      resolve({data: {jobId: '123'}});
    }, 500);
  });
};

export {analyze};
