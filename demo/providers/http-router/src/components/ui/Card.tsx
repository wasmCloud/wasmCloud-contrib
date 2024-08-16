import {cn} from '@/utils/cn';
import React, {HTMLProps} from 'react';

const Card = React.forwardRef<HTMLDivElement, HTMLProps<HTMLDivElement>>(
  ({children, className, ...props}, ref) => {
    return (
      <div
        ref={ref}
        className={cn(
          'bg-surface border-border-surface rounded-xl border shadow-sm py-6 px-8',
          className,
        )}
        {...props}
      >
        {children}
      </div>
    );
  },
);

export {Card};
