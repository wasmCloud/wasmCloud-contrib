import useDragging, {HTMLFileInputElement, HTMLFileInputProps} from '@/hooks/useDragging';
import {cn} from '@/utils/cn';
import React, {useImperativeHandle} from 'react';

type UploadProps = Omit<HTMLFileInputProps, 'onDrop'> & {
  onDrop: (file: File) => void;
};

const Upload = React.forwardRef<HTMLFileInputElement, UploadProps>(
  ({className, onDrop, ...props}, forwardedRef) => {
    const labelRef = React.useRef<HTMLLabelElement>(null);
    const inputRef = React.useRef<HTMLFileInputElement>(null);
    useImperativeHandle(forwardedRef, () => inputRef.current!);
    const hovered = useDragging({labelRef, inputRef, onDrop});

    return (
      <label
        ref={labelRef}
        className={cn(
          'block shadow-sm transition-colors bg-transparent hover:cursor-pointer',
          'rounded border-4 border-dashed border-primary-foreground/20',
          'hover:border-primary-foreground/40 hover:bg-primary-foreground/5',
          hovered && 'border-primary-foreground/60 bg-primary-foreground/10',
          className,
        )}
      >
        <input ref={inputRef} multiple={false} className="hidden" type="file" {...props} />
        <div className="block p-4 text-center">
          <p>
            <span className="text-accent underline">Select a file from your device</span>
          </p>
          <p>or drag and drop here</p>
        </div>
      </label>
    );
  },
);

export {Upload};
