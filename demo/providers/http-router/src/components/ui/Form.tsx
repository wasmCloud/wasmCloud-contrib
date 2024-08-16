import {cn} from '@/utils/cn';
import React, {HTMLProps} from 'react';
import {Heading} from './Heading';

type FormProps = Omit<HTMLProps<HTMLFormElement>, 'onSubmit'> & {
  onSubmit: (data: {url: string; width: number}) => void;
};

const Form = React.forwardRef<HTMLFormElement, FormProps>(({onSubmit, ...props}, ref) => {
  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const url = formData.get('url') as string;
    const width = Number(formData.get('width'));

    if (!url || !width) {
      return;
    }

    onSubmit({url, width});
  };

  return (
    <form ref={ref} {...props} onSubmit={handleSubmit}>
      <div className="text-center mb-8">
        <Heading as="h2">Load Provenance</Heading>
      </div>
      <div className="flex flex-col gap-8 items-start">
        <FormInput
          type="input"
          name="url"
          label="Origin URL"
          defaultValue="https://contentcredentials.org/_app/immutable/assets/home1.33703fa3.jpg"
        />
        <FormInput
          type="number"
          name="width"
          step="1"
          min={10}
          max={1024}
          label="Width"
          defaultValue="800"
        />
        <button type="submit" className="bg-primary text-white px-4 py-2 rounded-md">
          Submit
        </button>
      </div>
    </form>
  );
});

type InputProps = HTMLProps<HTMLInputElement> & {
  label: string;
};

const FormInput = React.forwardRef<HTMLInputElement, InputProps>(
  ({label, className, ...props}, ref) => {
    return (
      <label className="flex flex-col gap-1 w-full">
        <span className="text-sm font-bold uppercase">{label}</span>
        <input
          {...props}
          ref={ref}
          className={cn('bg-background border border-border-surface rounded px-2 py-1', className)}
        />
      </label>
    );
  },
);

export {Form};
