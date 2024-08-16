import React, {DetailedHTMLProps, HTMLAttributes} from 'react';

let draggingCount = 0;
export type HTMLFileInputElement = Omit<HTMLInputElement, 'multiple' | 'type'> & {
  multiple: false;
  type: 'file';
};
export type HTMLFileInputProps = Omit<
  DetailedHTMLProps<HTMLAttributes<HTMLInputElement>, HTMLFileInputElement>,
  'ref' | 'multiple' | 'type'
> & {
  ref?: React.Ref<HTMLFileInputElement>;
};

type UseDraggingOptions = {
  labelRef: React.RefObject<HTMLLabelElement>;
  inputRef: React.RefObject<HTMLFileInputElement>;
  onDrop?: (file: File) => void;
};

/**
 *
 * @param data - labelRef, inputRef, handleChanges, onDrop
 * @returns boolean - the state.
 *
 * @internal
 */
export default function useDragging({labelRef, inputRef, onDrop}: UseDraggingOptions): boolean {
  const [dragging, setDragging] = React.useState(false);
  const handleClick = React.useCallback(() => {
    inputRef.current?.click();
  }, [inputRef]);

  const handleDragIn = React.useCallback((event: DragEvent) => {
    event.preventDefault();
    event.stopPropagation();
    draggingCount++;
    if (event.dataTransfer?.items?.length !== 0) {
      setDragging(true);
    }
  }, []);
  const handleDragOut = React.useCallback((event: DragEvent) => {
    event.preventDefault();
    event.stopPropagation();
    draggingCount--;
    if (draggingCount > 0) return;
    setDragging(false);
  }, []);
  const handleDrag = React.useCallback((event: DragEvent) => {
    event.preventDefault();
    event.stopPropagation();
  }, []);
  const handleDrop = React.useCallback(
    (event: DragEvent) => {
      event.preventDefault();
      event.stopPropagation();
      setDragging(false);
      draggingCount = 0;

      const file = event.dataTransfer?.files?.[0];
      if (!file) return;

      onDrop?.(file);
    },
    [onDrop],
  );

  React.useEffect(() => {
    const label = labelRef.current;
    if (!label) return;
    label.addEventListener('click', handleClick);
    label.addEventListener('dragenter', handleDragIn);
    label.addEventListener('dragleave', handleDragOut);
    label.addEventListener('dragover', handleDrag);
    label.addEventListener('drop', handleDrop);
    return () => {
      label.removeEventListener('click', handleClick);
      label.removeEventListener('dragenter', handleDragIn);
      label.removeEventListener('dragleave', handleDragOut);
      label.removeEventListener('dragover', handleDrag);
      label.removeEventListener('drop', handleDrop);
    };
  }, [handleClick, handleDragIn, handleDragOut, handleDrag, handleDrop, labelRef]);

  return dragging;
}
