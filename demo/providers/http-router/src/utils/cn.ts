import cx from 'classix';
import {twMerge} from 'tailwind-merge';

function cn(...classes: (string | false | null | undefined)[]): string {
  return twMerge(cx(...classes));
}

export {cn};
