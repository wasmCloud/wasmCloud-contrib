import {HTMLProps} from 'react';
import {cn} from '@/utils/cn';
import {cva, VariantProps} from 'class-variance-authority';

type Props = HTMLProps<HTMLHeadingElement> & VariantProps<typeof heading>;

const heading = cva('', {
  variants: {
    as: {
      h1: 'text-6xl font-bold',
      h2: 'text-2xl font-bold',
      h3: 'text-xl font-bold',
      h4: 'text-lg font-bold',
      h5: 'text-base font-bold',
      h6: 'text-sm font-bold',
    },
  },
});

function Heading({children, as = 'h1', className, ...props}: Props) {
  const Component = as;
  return (
    <Component className={cn(heading({as}), className)} {...props}>
      {children}
    </Component>
  );
}

export {Heading};
