import {ConfigProvider} from '@/services/config/context';
import {ErrorBoundary} from '@/features/core/components/ErrorBoundary';
import {Loader} from '@/components/ui/Loader';
import React, {Suspense} from 'react';

function AppProvider({children}: React.PropsWithChildren) {
  return (
    <ErrorBoundary>
      <Suspense fallback={<Loader />}>
        <ConfigProvider>{children}</ConfigProvider>
      </Suspense>
    </ErrorBoundary>
  );
}

export {AppProvider};
