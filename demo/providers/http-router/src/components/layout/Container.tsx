import {cn} from '@/utils/cn';
import {HTMLProps} from 'react';

function Container({children, className}: HTMLProps<HTMLDivElement>) {
  return <div className={cn('container mx-auto px-4 max-w-screen-lg', className)}>{children}</div>;
}

export default Container;
